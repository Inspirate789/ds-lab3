FROM golang:1.23.1-bookworm AS build
WORKDIR /build
ENV CGO_ENABLED=0

# Install dependencies
COPY go.* .
RUN go mod download

# Get path to main.go
ARG MAIN_PATH

# Build the binary
# '--mount=target=.': use bind mounting from the build context
# '--mount=type=cache,target=/root/.cache/go-build': use go’s compiler cache
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    go build \
    -trimpath -ldflags "-s -w -extldflags '-static'" \
    -o /app $MAIN_PATH

FROM scratch AS app
# Add label to image
ARG PIPELINE_ID
LABEL version="$PIPELINE_ID"
# Copy the binary
COPY --from=build /app /app
# Get path to config and migrations
ARG CONFIG_PATH
ARG MIGRATIONS_FOLDER
# Create environment
COPY $MIGRATIONS_FOLDER /migrations
COPY $CONFIG_PATH /app.yaml
# Run the binary
ENTRYPOINT ["/app", "--config=/app.yaml"]
