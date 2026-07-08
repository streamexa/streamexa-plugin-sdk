package pluginsdk

import "github.com/lanyitin/streamexa-plugin-sdk/pluginpb"

// snapshotFromProto converts an incoming wire snapshot to the SDK type.
func snapshotFromProto(s *pluginpb.PageSnapshot) Snapshot {
	if s == nil {
		return Snapshot{}
	}
	out := Snapshot{PageDOM: s.GetPageDom()}
	for _, f := range s.GetIframes() {
		out.IFrames = append(out.IFrames, IFrame{FrameURL: f.GetFrameUrl(), DOM: f.GetDom()})
	}
	for _, r := range s.GetRequests() {
		out.Requests = append(out.Requests, Request{
			ID:            r.GetId(),
			Method:        r.GetMethod(),
			URL:           r.GetUrl(),
			Status:        int(r.GetStatus()),
			ResourceType:  r.GetResourceType(),
			Headers:       r.GetHeaders(),
			MediaType:     r.GetMediaType(),
			VideoDuration: r.GetVideoDuration(),
			IsAd:          r.GetIsAd(),
			IsLiveStream:  r.GetIsLiveStream(),
		})
	}
	for _, s := range s.GetStreamResults() {
		out.StreamResults = append(out.StreamResults, sourceFromProto(s))
	}
	return out
}

func sourceFromProto(s *pluginpb.AnalyzedSource) Source {
	return Source{
		URL:            s.GetUrl(),
		RequestHeaders: s.GetRequestHeaders(),
		MimeType:       s.GetMimeType(),
		VideoDuration:  s.GetVideoDuration(),
		EstimatedBytes: s.GetEstimatedBytes(),
	}
}

func sourceToProto(s Source) *pluginpb.AnalyzedSource {
	return &pluginpb.AnalyzedSource{
		Url:            s.URL,
		RequestHeaders: s.RequestHeaders,
		MimeType:       s.MimeType,
		VideoDuration:  s.VideoDuration,
		EstimatedBytes: s.EstimatedBytes,
	}
}

// resultToProto converts the plugin's Result to the wire result.
func resultToProto(r Result) *pluginpb.PluginResult {
	out := &pluginpb.PluginResult{Metadata: r.Metadata}
	for _, s := range r.Sources {
		out.Sources = append(out.Sources, sourceToProto(s))
	}
	if r.Task != nil {
		task := &pluginpb.PluginTask{Labels: r.Task.Labels}
		for _, s := range r.Task.Sources {
			task.Sources = append(task.Sources, sourceToProto(s))
		}
		out.Task = task
	}
	return out
}
