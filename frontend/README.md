# Mimir AIP Frontend

Web UI for the Mimir AIP orchestrator built with React and the 6 primitive components.

## Quick Start

### Local Development
```bash
# From project root
make dev-frontend

# Or directly
cd frontend
PORT=3000 API_URL=http://localhost:8080 go run server.go
```

Access at http://localhost:3000

### Docker Build
```bash
# From project root
docker build -t mimir-aip/frontend:latest -f frontend/Dockerfile .
```

### Docker Run
```bash
docker run -p 3000:3000 -e API_URL=http://localhost:8080 mimir-aip/frontend:latest
```

## Architecture

### Components
Built using only 6 primitive components as defined in the plan:
1. **Tabs** - Navigation and sub-sections
2. **Forms** - All input forms with validation
3. **Tables** - Data display with actions
4. **Buttons** - All interactive elements
5. **Modals** - Overlay dialogs
6. **Graph** - Chart.js integration (ready for metrics)

### Pages
1. **Projects** - CRUD operations for projects
2. **Pipelines** - Pipeline management + Recurring Jobs tab
3. **Ontologies** - Ontology management + Extraction from storage
4. **ML Models** - Model management + Training
5. **Digital Twins** - Twin management with Entities/Scenarios/Actions tabs
6. **Storage** - Storage configuration management
7. **Plugins** - Plugin installation and management
8. **Work Queue** - Real-time work task monitoring

### API Integration
- All API calls go through the Go server which proxies to orchestrator
- Automatic retry and error handling
- Environment-based API URL configuration

## Files

- `index.html` - HTML shell with React/Babel/Chart.js CDN imports
- `app.js` - Complete React application with all components and pages
- `styles.css` - Styling with defined color palette
- `server.go` - Go HTTP server with API proxy
- `Dockerfile` - Multi-stage build for containerization

## Styling

### Color Palette
```css
--background: #1a2236  /* Dark navy blue */
--accent: #ff9900      /* Orange highlights */
--text: #ffffff        /* White text */
--font-family: 'Google Sans Code', monospace
```

### Status Colors
- Active/Completed: Green
- Inactive: Gray
- Pending: Yellow
- Running/Training: Blue
- Failed: Red
- Deployed: Teal

## Environment Variables

- `PORT` - HTTP server port (default: 3000)
- `API_URL` - Orchestrator API URL (default: http://localhost:8080)

## Development

### Making Changes
1. Edit `app.js` for React components/pages
2. Edit `styles.css` for styling
3. Edit `server.go` for backend proxy logic
4. Refresh browser - changes are loaded on page refresh (no build needed)

### Adding a New Page
1. Create page component function in `app.js`
2. Add page to `pages` array in `App` component
3. Add case in `renderPage()` switch statement
4. Use existing primitive components only

### API Calls
Use the `apiCall` helper function:
```javascript
const data = await apiCall('/api/endpoint', {
  method: 'POST',
  body: JSON.stringify(payload)
});
```

## Production Deployment

See [DEPLOYMENT.md](../DEPLOYMENT.md) in the project root for complete deployment instructions.

### Quick Deploy
```bash
# Kubernetes
make build-frontend
make deploy-k8s

# Docker Compose
docker-compose up frontend
```

## Troubleshooting

### Frontend cannot connect to orchestrator
- Check `API_URL` environment variable
- Verify orchestrator is running and accessible
- Check browser console for CORS errors

### Changes not appearing
- Hard refresh browser (Cmd+Shift+R / Ctrl+Shift+R)
- Clear browser cache
- Check for JavaScript errors in browser console

### Build failures
- Ensure Go 1.21+ is installed
- Check Dockerfile paths are correct
- Verify all files are in place (index.html, app.js, styles.css, server.go)
