package error

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	log "github.com/pion/ion-log"
	ion "github.com/pion/ion/pkg/grpc/ion"
)

// err.NewGrpcIonError(codes.InvalidArgument, "GRPC error with custom error", -1, "custom error")
func NewGrpcIonError(code codes.Code, msg string, errorCode int32, desc string, debugging *ion.Debugging) error {
	st := status.New(code, msg)
	customErr := &ion.IonError{
		ErrorCode:   errorCode,
		Description: desc,
		Debugging:   debugging,
	}
	stDetails, _ := st.WithDetails(customErr)
	return stDetails.Err()
}

func ParseGrpcIonError(err error) (*ion.IonError, bool) {
	st, ok := status.FromError(err)
	if !ok {
		log.Errorf("Error: %v", err)
		return nil, false
	}

	log.Infof("GRPC Error : Code [%d], Message [%s]", st.Code(), st.Message())

	if len(st.Details()) > 0 {
		for _, detail := range st.Details() {
			switch d := detail.(type) {
			case *ion.IonError:
				log.Infof("  - Details: IonError: %d, %s", d.ErrorCode, d.Description)
				return d, true
			default:
				log.Infof("  - Details: Unknown: %v", d)
			}
		}
	}

	return nil, false
}
