package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/danielgtaylor/huma/v2/conditional"

	_ "embed"
)

//go:embed README.md
var apiDesc string

type PublishPoint struct {
	ID      string            `json:"id" example:"pub1" doc:"The unique identifier for the publish point."`
	Format  string            `json:"format" enum:"hls,dash" doc:"The format to publish in."`
	URL     string            `json:"url" format:"uri" doc:"The URL to publish to."`
	DRMs    []string          `json:"drms,omitempty" enum:"fairplay,widevine,playready" doc:"A list of DRM systems to use."`
	Headers map[string]string `json:"headers,omitempty" doc:"A map of headers to include in the request."`
}

type VideoEncoder struct {
	ID        string  `json:"id" example:"hd1" doc:"The unique identifier for the encoder."`
	Width     uint32  `json:"width" multipleOf:"2" example:"1920" doc:"The width of the video in pixels. Width & height must result in an aspect ratio of 16:9."`
	Height    uint32  `json:"height" multipleOf:"2" example:"1080" doc:"The height of the video in pixels. Width & height must result in an aspect ratio of 16:9."`
	Bitrate   uint16  `json:"bitrate" minimum:"300" doc:"The target bitrate for the video in kbps."`
	Framerate float64 `json:"framerate" enum:"30,25,29.97,50,60" doc:"The target framerate for the video in frames per second."`
}

func (v *VideoEncoder) Resolve(ctx huma.Context, prefix *huma.PathBuffer) []error {
	// Custom validation for the aspect ratio.
	if float64(v.Width)/float64(v.Height) != 16.0/9.0 {
		return []error{&huma.ErrorDetail{
			Message:  "width and height must be in a 16:9 (1.777) aspect ratio",
			Location: prefix.String(),
			Value:    float64(v.Width) / float64(v.Height),
		}}
	}
	return nil
}

type Channel struct {
	Name            string         `json:"name" maxLength:"80" doc:"The friendly name of the channel."`
	Region          string         `json:"region" enum:"us-west,us-east" doc:"The desired region to run the channel in."`
	On              bool           `json:"on,omitempty" doc:"Whether the channel is currently running."`
	SegmentDuration uint8          `json:"segment_duration" minimum:"2" maximum:"60" doc:"The duration of each video segment in seconds."`
	Tags            []string       `json:"tags,omitempty" maxItems:"10" example:"[\"event\", \"olympics\"]" doc:"A list of tags for the channel."`
	VideoEncoders   []VideoEncoder `json:"video_encoders" minItems:"1" doc:"A list of video encoder settings use."`
	PublishPoints   []PublishPoint `json:"publish_points,omitempty" doc:"A list of publishing points to use."`
}

// ChannelMeta is used both as the DB storage object as well as the response
// for listing channels.
type ChannelMeta struct {
	ID           string    `json:"id" doc:"Channel ID"`
	ETag         string    `json:"etag" doc:"The content hash for the channel"`
	LastModified time.Time `json:"last_modified" doc:"The last modified time for the channel"`
	Channel      *Channel  `json:"-"`
}

// ChannelIDParam is a shared input path parameter used by several operations.
type ChannelIDParam struct {
	ChannelID string `path:"id" pattern:"[a-zA-Z0-9_-]{2,60}" doc:"The unique identifier of the channel."`
}

type ListChannelsResponse struct {
	Link string `header:"Link" doc:"Links for pagination"`
	Body []*ChannelMeta
}

type GetChannelResponse struct {
	ETag         string    `header:"Etag" doc:"The content hash for the channel"`
	LastModified time.Time `header:"Last-Modified" doc:"The last modified time for the channel"`
	Body         *Channel
}

type PutChannelResponse struct {
	ETag string `header:"ETag" doc:"The content hash for the channel"`
}

// setup our API middleware, operations, and handlers.
func setup(api huma.API, db DB[*ChannelMeta]) {
	// Middleware example to log requests.
	api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		// Basic tracing support.
		traceID := GetTraceID()
		ctx = huma.WithValue(ctx, ctxKeyTraceID, traceID)
		ctx.SetHeader("traceparent", traceID)

		next(ctx)

		// Log the request.
		slog.Info("Request",
			"method", ctx.Method(),
			"path", ctx.URL().Path,
			"status", ctx.Status(),
			"trace_id", traceID,
		)
	})

	huma.Get(api, "/channels", func(ctx context.Context, input *struct {
		Cursor string `query:"cursor" doc:"The cursor to use for pagination."`
	}) (*ListChannelsResponse, error) {
		// TODO: pagination!
		metas := []*ChannelMeta{}
		db.Range(func(key string, value *ChannelMeta) bool {
			metas = append(metas, value)
			return true
		})
		sort.Slice(metas, func(i, j int) bool {
			// Bit of a hack due to the in-memory map, but let's make sure to send
			// clients a stable order of channels.
			return metas[i].LastModified.After(metas[j].LastModified)
		})
		return &ListChannelsResponse{
			Body: metas,
		}, nil
	})

	huma.Get(api, "/channels/{id}", func(ctx context.Context, input *struct {
		ChannelIDParam
	}) (*GetChannelResponse, error) {
		meta, ok := db.Load(input.ChannelID)
		if !ok {
			return nil, huma.Error404NotFound("Channel not found")
		}
		return &GetChannelResponse{
			ETag:         meta.ETag,
			LastModified: meta.LastModified,
			Body:         meta.Channel,
		}, nil
	})

	huma.Put(api, "/channels/{id}", func(ctx context.Context, input *struct {
		ChannelIDParam
		conditional.Params
		Body *Channel
	}) (*PutChannelResponse, error) {
		etag := ""
		modified := time.Time{}
		existing, ok := db.Load(input.ChannelID)
		if ok {
			etag = existing.ETag
			modified = existing.LastModified
		}
		if input.HasConditionalParams() {
			// Conditional update, so fail if the ETag/modified time doesn't match.
			// This prevents multiple distributed clients from overwriting each other.
			if err := input.PreconditionFailed(etag, modified); err != nil {
				return nil, err
			}
		}

		if existing != nil && Hash(input.Body) == Hash(existing.Channel) {
			return nil, huma.Status304NotModified()
		}

		meta := &ChannelMeta{
			ID:           input.ChannelID,
			ETag:         Hash(input.Body),
			LastModified: time.Now(),
			Channel:      input.Body,
		}
		db.Store(input.ChannelID, meta)

		return &PutChannelResponse{
			ETag: meta.ETag,
		}, nil
	})

	huma.Delete(api, "/channels/{id}", func(ctx context.Context, input *struct {
		ChannelIDParam
	}) (*struct{}, error) {
		db.Delete(input.ChannelID)
		return nil, nil
	})
}

func main() {
	// Create a new router & API
	router := http.NewServeMux()

	// Set up and create the API with some basic info.
	config := huma.DefaultConfig("Channel API Demo", "1.0.0")
	config.OpenAPI.Info.Description = strings.TrimPrefix(apiDesc, "# Huma Demo")
	config.OpenAPI.Info.Contact = &huma.Contact{
		Name:  "Channel API Support",
		Email: "support@example.com",
	}
	api := humago.New(router, config)

	// Initialize the DB. This is a goroutine-safe in-memory map for the demo,
	// but would be a real data store in a production system.
	db := NewDB[*ChannelMeta]("channels.db")

	// Register all our API operations & handlers.
	setup(api, db)

	// Run the server!
	fmt.Println("Listening on http://localhost:8888")
	err := http.ListenAndServe("localhost:8888", router)
	if err != http.ErrServerClosed {
		panic(err)
	}
}
