FROM golang:1.21-alpine3.19

# Set the working directory
WORKDIR /go/src

# Configure environment variables
ENV PATH="/go/bin:${PATH}"
ENV CGO_ENABLED=0

# Install Cobra CLI
RUN go install github.com/spf13/cobra-cli@latest

# Keep the container running for debugging
CMD ["tail", "-f", "/dev/null"]
