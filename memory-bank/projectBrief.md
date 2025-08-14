# Mycelian Memory - Project Brief

## Vision
Build a SaaS multi-agent memory management system that enables AI assistants and autonomous agents to maintain persistent, searchable context across conversations and sessions.

## Business Model
- **SaaS-First**: Focus on first paying customers, not open source
- **AWS-Native**: Deploy to AWS infrastructure for beta stack
- **Deployment Strategy**: Manual beta deployment, lightweight CI automation before public launch

## Core Requirements
- **Persistent Memory**: Store and retrieve contextual information across AI assistant sessions
- **Multi-Agent Support**: Enable multiple agents to work collaboratively with shared and private memory spaces
- **Vector Search**: Provide semantic search capabilities for memory retrieval
- **MCP Integration**: Implement Model Context Protocol for standardized AI tool integration
- **Simplified Architecture**: Single PostgreSQL dialect for focus and simplicity

## Key Components
1. **Memory Service API**: RESTful backend for memory management (port 11545)
2. **Go Client SDK**: Type-safe client library for memory operations
3. **MCP Server**: Tool server implementing Model Context Protocol
4. **CLI Tools**: mycelianCli for management, mycelian-service-tools for operations
5. **AWS Deployment Package**: Manual deployment code for beta stack

## Success Criteria
- First paying customers using the service
- AI assistants can maintain context across sessions
- Multi-agent collaboration without data corruption
- Sub-second search response times
- Production-ready reliability and security on AWS

## Project Status
- **Current Phase**: SaaS deployment preparation
- **Architecture**: Simplified single-database architecture
- **Deployment**: PostgreSQL (local dev), Aurora Serverless V2 (AWS prod)
- **Vector Search**: Evaluating Weaviate vs OpenSearch for AWS