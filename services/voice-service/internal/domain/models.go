package domain

type VoiceTokenResponse struct {
	LiveKitURL string `json:"livekit_url"`
	Token      string `json:"token"`
}

type LiveKitWebhookRequest struct {
	Event       string                 `json:"event"`
	Room        LiveKitRoom            `json:"room"`
	Participant LiveKitParticipant     `json:"participant"`
}

type LiveKitRoom struct {
	Name string `json:"name"`
	SID  string `json:"sid"`
}

type LiveKitParticipant struct {
	Identity string `json:"identity"`
	State    string `json:"state"`
	JoinedAt int64  `json:"joined_at"`
}
