package response

import "github.com/hllkk/devops-admin/server/model/media"

type FileUploadAndDownloadResponse struct {
	File media.FileUploadAndDownload `json:"file"`
}
