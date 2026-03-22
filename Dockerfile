FROM chromedp/headless-shell:stable

LABEL org.opencontainers.image.source="https://github.com/felixgeelhaar/scout"
LABEL org.opencontainers.image.description="AI-powered browser automation MCP server"
LABEL org.opencontainers.image.licenses="MIT"
LABEL io.modelcontextprotocol.server.name="io.github.felixgeelhaar/scout"

COPY scout /usr/local/bin/scout

ENTRYPOINT ["scout"]
CMD ["mcp", "serve"]
