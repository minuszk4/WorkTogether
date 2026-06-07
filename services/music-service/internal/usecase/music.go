package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/worktogether/services/music-service/internal/domain"
	"github.com/worktogether/services/music-service/internal/repository"
)

type MusicUsecase struct {
	postgresRepo *repository.PostgresRepository
	minioClient  *minio.Client
	minioBucket  string
	minioPublic  string // Public access endpoint for URLs
}

func NewMusicUsecase(pg *repository.PostgresRepository, minioEndpoint, accessKey, secretKey, bucket, minioPublic string) *MusicUsecase {
	// Khởi tạo MinIO client
	var mClient *minio.Client
	var err error
	for i := 0; i < 5; i++ {
		mClient, err = minio.New(minioEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: false,
		})
		if err == nil {
			break
		}
		log.Printf("Đang thử kết nối lại với MinIO (%d/5): %v\n", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Printf("Cảnh báo: Không thể khởi tạo MinIO Client: %v\n", err)
	} else {
		// Tạo bucket nếu chưa có
		ctx := context.Background()
		exists, err := mClient.BucketExists(ctx, bucket)
		if err == nil && !exists {
			err = mClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
			if err != nil {
				log.Printf("Lỗi tạo bucket %s: %v\n", bucket, err)
			} else {
				// Cấu hình policy sang public-read cho bucket
				policy := fmt.Sprintf(`{
					"Version": "2012-10-17",
					"Statement": [
						{
							"Effect": "Allow",
							"Principal": {"AWS": ["*"]},
							"Action": ["s3:GetObject"],
							"Resource": ["arn:aws:s3:::%s/*"]
						}
					]
				}`, bucket)
				err = mClient.SetBucketPolicy(ctx, bucket, policy)
				if err != nil {
					log.Printf("Lỗi đặt public policy cho bucket: %v\n", err)
				}
				log.Printf("Đã tạo thành công bucket %s với public read policy.\n", bucket)
			}
		}
	}

	return &MusicUsecase{
		postgresRepo: pg,
		minioClient:  mClient,
		minioBucket:  bucket,
		minioPublic:  minioPublic,
	}
}

func (u *MusicUsecase) ExtractYoutubeMetadata(ctx context.Context, videoURL string) (*domain.Track, error) {
	// Kiểm tra xem track đã tồn tại trong DB chưa
	existing, err := u.postgresRepo.GetTrackBySourceURL(ctx, videoURL)
	if err == nil && existing != nil {
		return existing, nil
	}

	// 1. Gọi oembed để lấy title, artist, thumbnail
	oembedURL := fmt.Sprintf("https://www.youtube.com/oembed?url=%s&format=json", url.QueryEscape(videoURL))
	resp, err := http.Get(oembedURL)
	if err != nil {
		return nil, fmt.Errorf("không thể kết nối dịch vụ YouTube: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("không tìm thấy thông tin video từ YouTube")
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("lỗi giải mã dữ liệu YouTube: %v", err)
	}

	title, _ := data["title"].(string)
	artist, _ := data["author_name"].(string)
	thumbnail, _ := data["thumbnail_url"].(string)

	// 2. Scrape trang chính để lấy độ dài video (duration_ms)
	durationMS := 180000 // Fallback: 3 phút
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", videoURL, nil)
	if err == nil {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		pageResp, err := client.Do(req)
		if err == nil {
			defer pageResp.Body.Close()
			htmlBytes, err := io.ReadAll(pageResp.Body)
			if err == nil {
				html := string(htmlBytes)
				// Tìm lengthSeconds
				re := regexp.MustCompile(`"lengthSeconds":"(\d+)"`)
				match := re.FindStringSubmatch(html)
				if len(match) > 1 {
					sec, _ := strconv.Atoi(match[1])
					durationMS = sec * 1000
				} else {
					// Thử approxDurationMs
					reApprox := regexp.MustCompile(`"approxDurationMs":"(\d+)"`)
					matchApprox := reApprox.FindStringSubmatch(html)
					if len(matchApprox) > 1 {
						ms, _ := strconv.Atoi(matchApprox[1])
						durationMS = ms
					}
				}
			}
		}
	}

	track := &domain.Track{
		ID:           uuid.New().String(),
		Title:        title,
		Artist:       artist,
		ThumbnailURL: thumbnail,
		DurationMS:   durationMS,
		Source:       "youtube",
		SourceURL:    videoURL,
		CreatedAt:    time.Now(),
	}

	// Lưu vào DB
	if err := u.postgresRepo.SaveTrack(ctx, track); err != nil {
		return nil, err
	}

	return track, nil
}

func (u *MusicUsecase) ExtractTrackMetadata(ctx context.Context, sourceURL string) (*domain.Track, error) {
	// Kiểm tra xem track đã tồn tại trong DB chưa
	existing, err := u.postgresRepo.GetTrackBySourceURL(ctx, sourceURL)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Phân tích domain của sourceURL
	uParsed, err := url.Parse(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("URL không hợp lệ: %v", err)
	}

	host := strings.ToLower(uParsed.Host)
	if strings.Contains(host, "youtube.com") || strings.Contains(host, "youtu.be") {
		return u.ExtractYoutubeMetadata(ctx, sourceURL)
	} else if strings.Contains(host, "soundcloud.com") {
		return u.ExtractSoundCloudMetadata(ctx, sourceURL)
	}

	// Mặc định đối với các nền tảng/URL audio tự do khác
	return u.ExtractGenericMetadata(ctx, sourceURL)
}

func (u *MusicUsecase) ExtractSoundCloudMetadata(ctx context.Context, trackURL string) (*domain.Track, error) {
	oembedURL := fmt.Sprintf("https://soundcloud.com/oembed?url=%s&format=json", url.QueryEscape(trackURL))
	
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", oembedURL, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("không thể kết nối dịch vụ SoundCloud: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("không tìm thấy thông tin bài hát từ SoundCloud (Mã lỗi: %d)", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("lỗi giải mã dữ liệu SoundCloud: %v", err)
	}

	title, _ := data["title"].(string)
	artist, _ := data["author_name"].(string)
	thumbnail, _ := data["thumbnail_url"].(string)

	if title == "" {
		title = "SoundCloud Track"
	}
	if artist == "" {
		artist = "SoundCloud Artist"
	}

	track := &domain.Track{
		ID:           uuid.New().String(),
		Title:        title,
		Artist:       artist,
		ThumbnailURL: thumbnail,
		DurationMS:   180000, // Fallback 3 phút
		Source:       "soundcloud",
		SourceURL:    trackURL,
		CreatedAt:    time.Now(),
	}

	if err := u.postgresRepo.SaveTrack(ctx, track); err != nil {
		return nil, err
	}

	return track, nil
}

func (u *MusicUsecase) ExtractGenericMetadata(ctx context.Context, trackURL string) (*domain.Track, error) {
	title := "Bài hát tự do"
	uParsed, err := url.Parse(trackURL)
	if err == nil {
		pathParts := strings.Split(uParsed.Path, "/")
		if len(pathParts) > 0 && pathParts[len(pathParts)-1] != "" {
			fileName := pathParts[len(pathParts)-1]
			if dotIdx := strings.LastIndex(fileName, "."); dotIdx != -1 {
				fileName = fileName[:dotIdx]
			}
			if unescaped, err := url.PathUnescape(fileName); err == nil {
				title = unescaped
			} else {
				title = fileName
			}
		}
	}

	track := &domain.Track{
		ID:           uuid.New().String(),
		Title:        title,
		Artist:       "Nguồn tự do",
		ThumbnailURL: "",
		DurationMS:   180000, // Fallback 3 phút
		Source:       "generic",
		SourceURL:    trackURL,
		CreatedAt:    time.Now(),
	}

	if err := u.postgresRepo.SaveTrack(ctx, track); err != nil {
		return nil, err
	}

	return track, nil
}

func (u *MusicUsecase) UploadAudioFile(ctx context.Context, reader io.Reader, fileSize int64, originalFilename, title, artist string) (*domain.Track, error) {
	if u.minioClient == nil {
		return nil, fmt.Errorf("minio client chưa được khởi tạo")
	}

	// Sinh tên file ngẫu nhiên bảo mật
	ext := "mp3"
	if parts := strings.Split(originalFilename, "."); len(parts) > 1 {
		ext = parts[len(parts)-1]
	}
	filename := fmt.Sprintf("%s.%s", uuid.New().String(), ext)

	// Upload lên MinIO
	_, err := u.minioClient.PutObject(ctx, u.minioBucket, filename, reader, fileSize, minio.PutObjectOptions{
		ContentType: "audio/mpeg",
	})
	if err != nil {
		return nil, fmt.Errorf("lỗi tải file lên MinIO: %v", err)
	}

	// Tạo URL công khai
	sourceURL := fmt.Sprintf("%s/%s/%s", u.minioPublic, u.minioBucket, filename)

	if title == "" {
		title = originalFilename
	}
	if artist == "" {
		artist = "Uploaded Music"
	}

	track := &domain.Track{
		ID:           uuid.New().String(),
		Title:        title,
		Artist:       artist,
		ThumbnailURL: "", // Có thể bổ sung thumbnail mặc định
		DurationMS:   180000, // MP3 upload giả định 3 phút hoặc cần thư viện đọc audio header
		Source:       "upload",
		SourceURL:    sourceURL,
		CreatedAt:    time.Now(),
	}

	// Lưu vào DB
	if err := u.postgresRepo.SaveTrack(ctx, track); err != nil {
		return nil, err
	}

	return track, nil
}

func (u *MusicUsecase) GetTrack(ctx context.Context, id string) (*domain.Track, error) {
	return u.postgresRepo.GetTrackByID(ctx, id)
}

func (u *MusicUsecase) SearchTracks(ctx context.Context, keyword string) ([]*domain.Track, error) {
	if keyword == "" {
		return u.postgresRepo.SearchTracks(ctx, keyword, 20)
	}

	// Scrape YouTube search results dynamically
	results, err := u.SearchYoutubeScrape(ctx, keyword)
	if err != nil {
		log.Printf("Warning: YouTube scrape failed: %v, falling back to database search\n", err)
		return u.postgresRepo.SearchTracks(ctx, keyword, 20)
	}

	return results, nil
}

func (u *MusicUsecase) SearchYoutubeScrape(ctx context.Context, keyword string) ([]*domain.Track, error) {
	searchURL := fmt.Sprintf("https://www.youtube.com/results?search_query=%s&sp=EgIQAQ%%253D%%253D", url.QueryEscape(keyword))

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	// YouTube requires a modern browser User-Agent to return desktop layout with ytInitialData
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("YouTube returned status code %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	html := string(bodyBytes)

	// Find ytInitialData JSON
	idx := strings.Index(html, "ytInitialData = ")
	if idx == -1 {
		idx = strings.Index(html, "ytInitialData =")
	}
	if idx == -1 {
		return nil, fmt.Errorf("could not find ytInitialData")
	}

	// Extract JSON block (find first '{' and match with matching closing '}')
	startIdx := strings.Index(html[idx:], "{")
	if startIdx == -1 {
		return nil, fmt.Errorf("could not find JSON start")
	}
	startIdx += idx

	// Simple bracket matching to extract the JSON block
	bracketCount := 0
	endIdx := -1
	for i := startIdx; i < len(html); i++ {
		if html[i] == '{' {
			bracketCount++
		} else if html[i] == '}' {
			bracketCount--
			if bracketCount == 0 {
				endIdx = i + 1
				break
			}
		}
	}

	if endIdx == -1 {
		return nil, fmt.Errorf("could not find JSON end")
	}

	jsonStr := html[startIdx:endIdx]

	// We use regex to match videoRenderer entries in the JSON string
	videoRendererRegex := regexp.MustCompile(`"videoRenderer":\{(.*?)\}(?:,"searchVideoResultEntityKey"|,"trackingParams")`)
	matches := videoRendererRegex.FindAllStringSubmatch(jsonStr, -1)

	var tracks []*domain.Track

	videoIdRegex := regexp.MustCompile(`"videoId":"([^"]+)"`)
	titleRegex := regexp.MustCompile(`"title":\{"runs":\[\{"text":"([^"]+)"\}`)
	channelRegex := regexp.MustCompile(`"ownerText":\{"runs":\[\{"text":"([^"]+)"\}`)
	thumbnailRegex := regexp.MustCompile(`"thumbnail":\{"thumbnails":\[\{"url":"([^"]+)"`)
	durationRegex := regexp.MustCompile(`"lengthText":\{"accessibility":\{"accessibilityData":\{"label":"[^"]+"\}\},"simpleText":"([^"]+)"\}`)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		content := match[1]

		idMatch := videoIdRegex.FindStringSubmatch(content)
		if len(idMatch) < 2 {
			continue
		}
		videoID := idMatch[1]

		title := "YouTube Video"
		titleMatch := titleRegex.FindStringSubmatch(content)
		if len(titleMatch) >= 2 {
			title = u.unescapeJSONString(titleMatch[1])
		}

		artist := "YouTube Artist"
		channelMatch := channelRegex.FindStringSubmatch(content)
		if len(channelMatch) >= 2 {
			artist = u.unescapeJSONString(channelMatch[1])
		}

		thumbnail := fmt.Sprintf("https://img.youtube.com/vi/%s/hqdefault.jpg", videoID)
		thumbMatch := thumbnailRegex.FindStringSubmatch(content)
		if len(thumbMatch) >= 2 {
			thumbnail = u.unescapeJSONString(thumbMatch[1])
		}

		durationMS := 180000 // default 3 mins
		durationMatch := durationRegex.FindStringSubmatch(content)
		if len(durationMatch) >= 2 {
			durationStr := durationMatch[1] // e.g. "4:30" or "1:15:30"
			parts := strings.Split(durationStr, ":")
			seconds := 0
			if len(parts) == 2 {
				m, _ := strconv.Atoi(parts[0])
				s, _ := strconv.Atoi(parts[1])
				seconds = m*60 + s
			} else if len(parts) == 3 {
				h, _ := strconv.Atoi(parts[0])
				m, _ := strconv.Atoi(parts[1])
				s, _ := strconv.Atoi(parts[2])
				seconds = h*3600 + m*60 + s
			}
			if seconds > 0 {
				durationMS = seconds * 1000
			}
		}

		tracks = append(tracks, &domain.Track{
			ID:           videoID,
			Title:        title,
			Artist:       artist,
			ThumbnailURL: thumbnail,
			DurationMS:   durationMS,
			Source:       "youtube",
			SourceURL:    "https://www.youtube.com/watch?v=" + videoID,
		})

		if len(tracks) >= 8 { // Return top 8 results
			break
		}
	}

	return tracks, nil
}

func (u *MusicUsecase) unescapeJSONString(s string) string {
	s = strings.ReplaceAll(s, `\u0026`, "&")
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	s = strings.ReplaceAll(s, `\/`, `/`)
	return s
}

func (u *MusicUsecase) LogPlayback(ctx context.Context, roomID, trackID string) error {
	h := &domain.PlaybackHistory{
		ID:       uuid.New().String(),
		RoomID:   roomID,
		TrackID:  trackID,
		PlayedAt: time.Now(),
	}
	return u.postgresRepo.SavePlaybackHistory(ctx, h)
}

func (u *MusicUsecase) GetRoomHistory(ctx context.Context, roomID string) ([]*domain.Track, error) {
	return u.postgresRepo.GetPlaybackHistory(ctx, roomID, 20)
}
