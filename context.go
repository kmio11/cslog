package cslog

import "context"

type (
	ctxKeyLogID       struct{}
	ctxKeyParentLogID struct{}
)

func GetLogID(ctx context.Context) LogID {
	if logID, ok := ctx.Value(ctxKeyLogID{}).(LogID); ok {
		return logID
	}
	return nil
}

func SetLogID(ctx context.Context, logID LogID) context.Context {
	return context.WithValue(ctx, ctxKeyLogID{}, logID)
}

func GetParentLogID(ctx context.Context) LogID {
	if logID, ok := ctx.Value(ctxKeyParentLogID{}).(LogID); ok {
		return logID
	}
	return nil
}

func SetParentLogID(ctx context.Context, parentLogID LogID) context.Context {
	return context.WithValue(ctx, ctxKeyParentLogID{}, parentLogID)
}

// WithLogContext returns a new context with a newly generated logId.
// If the given context already contains a logId, it is replaced with the new logId.
func WithLogContext(ctx context.Context) context.Context {
	return SetLogID(ctx, logIdGenerator.NewID())
}

// WithChildLogContext returns a new context with a newly generated logId.
// If the given context already contains a logId, it is set as the parentLogId.
func WithChildLogContext(ctx context.Context) context.Context {
	newParentId := GetLogID(ctx)
	newLogId := logIdGenerator.NewID()

	newCtx := SetParentLogID(ctx, newParentId)
	newCtx = SetLogID(newCtx, newLogId)

	return newCtx
}

// function for ContextAttr.getFn
func getLogIdFunc(ctx context.Context) (value string, ok bool) {
	logId := GetLogID(ctx)
	if logId == nil {
		return "", false
	}
	return logId.String(), !logId.IsZero()
}

// function for ContextAttr.getFn
func getParentLogIdFunc(ctx context.Context) (value string, ok bool) {
	parentLogId := GetParentLogID(ctx)
	if parentLogId == nil {
		return "", false
	}
	return parentLogId.String(), !parentLogId.IsZero()
}
