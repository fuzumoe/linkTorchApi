# Crawler Integration with Service and Handler Layers

## Overview

This document outlines how the enhanced crawler with Go channels is integrated with the rest of the URL Insight application. The integration ensures that the new crawler features are accessible through the service and API layers.

## Integration Points

### 1. URL Service Integration

The URL service acts as an intermediary between the handlers and the crawler pool:

```go
type URLService interface {
    // ... existing methods
    StartWithPriority(id uint, priority int) error
    GetCrawlResults() <-chan crawler.CrawlResult
    AdjustCrawlerWorkers(action string, count int) error
}
```

These methods expose the enhanced crawler capabilities:

- `StartWithPriority`: Allows starting a crawl with a specific priority (1-10)
- `GetCrawlResults`: Returns the channel that emits real-time crawl results
- `AdjustCrawlerWorkers`: Dynamically adds or removes workers from the crawler pool

### 2. REST API Integration

New API endpoints expose the enhanced crawler capabilities:

- `PATCH /urls/{id}/start?priority=N`: Start crawling with priority
- `PATCH /crawler/workers?action=add&count=N`: Add N workers to the pool
- `PATCH /crawler/workers?action=remove&count=N`: Remove N workers from the pool
- `GET /crawler/results`: Placeholder for real-time results (would typically be WebSocket)

### 3. Priority Levels

Priority levels range from 1 to 10:

- **8-10**: High priority (uses highPriority channel)
- **3-7**: Normal priority (uses normalPriority channel)
- **1-2**: Low priority (uses lowPriority channel)

### 4. Result Handling

The crawler emits results through the `results` channel which contains:

```go
type CrawlResult struct {
    URLID     uint
    URL       string
    Status    string
    Error     error
    LinkCount int
    Duration  time.Duration
    Links     []model.Link
}
```

This enables real-time monitoring and feedback on the crawling process.

## Usage Examples

### Starting a URL crawl with priority

```
PATCH /api/v1/urls/123/start?priority=8
```

Response:
```json
{
  "status": "queued",
  "priority": 8
}
```

### Adjusting crawler workers

```
PATCH /api/v1/crawler/workers?action=add&count=5
```

Response:
```json
{
  "message": "Successfully added 5 workers"
}
```

## Future Enhancements

1. **WebSocket Integration**: Implement a WebSocket endpoint for real-time crawl results
2. **Admin Dashboard**: Create a dashboard for monitoring crawler performance
3. **Scheduled Crawling**: Allow URLs to be scheduled for crawling at specific times
4. **Domain-specific Rate Limiting**: Implement per-domain crawl rate limiting
5. **Result Filtering**: Filter crawl results by status, time, or other criteria
