package response

import "github.com/hllkk/devops-admin/server/model/media"

type UploadInitResp struct {
	Instant        bool                         `json:"instant"`
	UploadID       uint                         `json:"uploadId"`
	UploadedChunks []int                        `json:"uploadedChunks"`
	Media          *media.FileUploadAndDownload `json:"media,omitempty"`
}

type UploadCompleteResp struct {
	Media media.FileUploadAndDownload `json:"media"`
}
