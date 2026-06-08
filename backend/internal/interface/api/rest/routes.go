package rest

// Route templates. Constants are not variables and are allowed.
const (
	routeAPIV1 = "/api/v1"

	routeAuth  = routeAPIV1 + "/auth"
	routeLogin = routeAuth + "/login"

	routePhotos      = routeAPIV1 + "/photos"
	routePhoto       = routePhotos + "/:photoId"
	routeUploadLink  = routePhoto + "/upload-link"
	routePhotoObject = routePhoto + "/object"
	routePhotoThumb  = routePhoto + "/thumbnail"
	routePhotoOrig   = routePhoto + "/original"

	routeHealth    = "/healthz"
	routeMetrics   = "/metrics"
	routeDebugVars = "/debug/vars"
)
