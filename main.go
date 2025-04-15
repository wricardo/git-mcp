package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

// MCPServer represents our MCP server instance
type MCPServer struct {
	server  *server.MCPServer
	workdir string
}

func (s *MCPServer) init() error {
	s.server = server.NewMCPServer("git-mcp", "0.1.0",
		server.WithResourceCapabilities(false, true),
		server.WithLogging(),
	)

	// configs
	s.workdir = os.Getenv("WORKDIR")
	if s.workdir == "" {
		return fmt.Errorf("WORKDIR env variable is not set")
	}

	// register tools
	s.registerTools()

	return nil
}

// Serve starts the MCP server over stdio
func (s *MCPServer) ServeStdio() error {
	return server.ServeStdio(s.server)
}

func main() {
	s := &MCPServer{}
	if err := s.init(); err != nil {
		log.Fatal("[git-mcp] Error:", err)
	}

	if err := s.ServeStdio(); err != nil {
		log.Fatal("[git-mcp] Error:", err)
	}
}
