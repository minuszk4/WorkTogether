package domain

type VoiceTokenResponse struct {
	LiveKitURL string `json:"livekit_url"`
	Token      string `json:"token"`
}

type LiveKitWebhookRequest struct {
	ID          string             `json:"id"`
	Event       string             `json:"event"`
	CreatedAt   int64              `json:"created_at"`
	Room        LiveKitRoom        `json:"room"`
	Participant LiveKitParticipant `json:"participant"`
}

type LiveKitRoom struct {
	Name string `json:"name"`
	SID  string `json:"sid"`
}

type LiveKitParticipant struct {
	Identity string `json:"identity"`
	SID      string `json:"sid"`
	State    string `json:"state"`
	JoinedAt int64  `json:"joined_at"`
}
