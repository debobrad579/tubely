# Tubely

A video sharing platform built to learn and implement file storage using AWS S3 and CloudFront CDN distribution. This project demonstrates modern cloud-based file handling, video processing, and content delivery.

## Overview

Tubely is a video upload and management application that allows users to create accounts, upload videos with thumbnails, and stream content through a CloudFront CDN. The application handles video processing (including aspect ratio detection and fast-start optimization) and stores processed videos in AWS S3 for scalable, global content delivery.

## Key Features

- **Video Management**: Create video drafts with titles and descriptions
- **Thumbnail Uploads**: Support for JPEG and PNG image formats
- **Video Processing**: 
  - Automatic aspect ratio detection (landscape/portrait)
  - Fast-start MP4 optimization using FFmpeg
  - Support for MP4 video format
- **Cloud Storage**: AWS S3 integration for scalable video storage
- **CDN Distribution**: CloudFront integration for fast, global content delivery
- **Local Asset Storage**: Thumbnails stored locally with filesystem management

## Architecture Highlights

### Video Upload Flow (`handler_upload_video.go`)

1. **Upload Reception**: Accepts video file uploads up to 1GB
2. **Validation**: Verifies user authentication and video ownership
3. **Temporary Storage**: Saves uploaded file temporarily for processing
4. **Aspect Ratio Detection**: Uses FFprobe to analyze video dimensions
5. **Fast-Start Processing**: Optimizes MP4 files with FFmpeg's `faststart` flag for progressive download
6. **S3 Upload**: Uploads processed video to S3 bucket organized by aspect ratio
7. **CloudFront URL**: Generates CDN URL for efficient content delivery
8. **Database Update**: Stores CloudFront URL in database for quick retrieval

### File Organization

Videos are organized in S3 by aspect ratio:
- `landscape/` - 16:9 videos
- `portrait/` - 9:16 videos  
- `other/` - Videos with different ratios

## Setup

### Prerequisites

- Go 1.23+
- FFmpeg and FFprobe (in PATH)
- SQLite 3
- AWS CLI configured with credentials
- AWS S3 bucket
- AWS CloudFront distribution

### Installation

1. Clone the repository
```bash
git clone <repository-url>
cd tubely
```

2. Install Go dependencies
```bash
go mod download
```

3. Download sample files (optional)
```bash
./samplesdownload.sh
```

4. Configure environment variables
```bash
cp .env.example .env
```

Edit `.env` with your configuration:
```
DB_PATH=tubely.db
JWT_SECRET=your-secret-key
PLATFORM=dev
FILEPATH_ROOT=./app
ASSETS_ROOT=./assets
S3_BUCKET=your-bucket-name
S3_REGION=us-east-1
S3_CF_DISTRO=your-cloudfront-domain.cloudfront.net
PORT=8091
```

## Learning Outcomes

This project demonstrates:

- **S3 Integration**: Direct file uploads to AWS S3 using the AWS SDK for Go
- **CDN Usage**: Serving content through CloudFront for improved performance
- **Video Processing**: FFmpeg integration for format optimization
- **Cloud Architecture**: Organizing and managing files in cloud storage
- **Scalable File Storage**: Moving from local filesystem to cloud-based solutions

## License

This project is part of the Boot.dev curriculum.
