# Stage 1: Build the Go binary
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /terraform-graphx .

# Stage 2: Create the final image
FROM hashicorp/terraform:latest

# Copy the built binary from the builder stage into the final image's PATH
COPY --from=builder /terraform-graphx /bin/terraform-graphx

# Set the working directory for user's Terraform files
WORKDIR /data

# The default entrypoint is `terraform`. Users can run the command like:
# docker run <imagename> graphx
# For example:
# docker build -t terraform-graphx-image .
# docker run --rm -v $(pwd)/examples:/data terraform-graphx-image graphx
ENTRYPOINT ["terraform"]
CMD ["graphx"]