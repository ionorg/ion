package error

type Code uint32

const (
	/*
		Canceled         = grpc.Canceled
		Unauthenticated  = grpc.Unauthenticated
		Unavailable      = grpc.Unavailable
		Unimplemented    = grpc.Unimplemented
		Internal         = grpc.Internal
		InvalidArgument  = grpc.InvalidArgument
		PermissionDenied = grpc.PermissionDenied
	*/

	Ok                     Code = 200
	BadRequest             Code = 400
	Forbidden              Code = 403
	NotFound               Code = 404
	RequestTimeout         Code = 408
	UnsupportedMediaType   Code = 415
	BusyHere               Code = 486
	TemporarilyUnavailable Code = 480
	InternalError          Code = 500
	NotImplemented         Code = 501
	ServiceUnavailable     Code = 503
)
