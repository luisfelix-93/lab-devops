# PR Summary

## 🚀 New Features

### 📚 Learning Tracks Support
Introduced a new "Tracks" system to organize labs into structured learning paths.
- **Database Schema**:
    - Created `tracks` table.
    - Updated `labs` table with `track_id` and `lab_order`.
- **API**:
    - Added endpoints to Create and List Tracks.
    - Updated Lab creation to support assignment to a Track.

### ⚡ Dockerfile Optimization
- Implemented **Docker BuildKit Cache Mounts** to significantly reduce build times.
- Added caching for Go modules (`/go/pkg/mod`) and build artifacts (`/root/.cache/go-build`).

## 🛠️ Improvements
- **Database**: Enhanced `workspaces` table to track execution status (`in_progress` vs `completed`).
- **Codebase**: Refactored `handler.go` and `lab_service.go` to support the new Tracks domain logic.

## 📝 Documentation
- Updated README.md with new features and instructions for using the Tracks system.
- Added documentation for the new API endpoints in the API documentation.
