package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

// 1. 구조체 정의 (JSON 데이터 파싱 및 Go 내부용 데이터 분리)
type Post struct {
	PageID     string `json:"page_id"`
	Title      string `json:"title"`
	CreatedAt  string `json:"created_at"`
	LastEdited string `json:"last_edited"`
	URL        string `json:"url"`
	Content    string `json:"content"`
	Category   string `json:"-"` // JSON에는 없지만 Go에서 사용할 카테고리
	CleanTitle string `json:"-"` // JSON에는 없지만 Go에서 사용할 실제 제목
}

// 2. 공통 데이터 처리 함수 (JSON을 읽어 카테고리와 제목을 쪼개는 역할)
func getProcessedPosts() []Post {
	// 데이터 경로가 다를 경우 "data/r3lis_posts.json" 부분을 수정해주세요.
	byteValue, err := os.ReadFile("/data/r3lis_posts.json")
	if err != nil {
		log.Println("JSON 데이터 읽기 실패:", err)
		return []Post{}
	}

	var posts []Post
	if err := json.Unmarshal(byteValue, &posts); err != nil {
		log.Println("JSON 파싱 실패:", err)
		return []Post{}
	}

	// 모든 글을 순회하며 "카테고리 :: 실제제목" 형태를 분리합니다.
	for i := range posts {
		parts := strings.Split(posts[i].Title, "::")
		if len(parts) >= 2 {
			posts[i].Category = strings.TrimSpace(parts[0])   // 예: 기획
			posts[i].CleanTitle = strings.TrimSpace(parts[1]) // 예: Aether Break
		} else {
			posts[i].Category = "기타"
			posts[i].CleanTitle = strings.TrimSpace(posts[i].Title)
		}
	}
	return posts
}

func main() {
	// (선택) 정적 파일 서빙이 필요하시다면 아래 주석을 해제하여 사용하세요.
	// http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// ==========================================
	// [라우터 1] 메인 화면 ("/")
	// ==========================================
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		posts := getProcessedPosts() // 분리 작업이 완료된 전체 데이터 호출

		// 메인 템플릿 파일 이름은 사용자님의 환경에 맞게 변경해주세요. (예: index.html)
		tmpl, err := template.ParseFiles("index.html", "sidebar.html")
		if err != nil {
			log.Println("🚨 템플릿 로드 실패 원인:", err) // <--- 이 줄을 추가
			http.Error(w, "메인 템플릿 에러\n", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, posts) // 메인 화면에는 전체 글 전달
	})

	// ==========================================
	// [라우터 2] 블로그 필터링 화면 ("/blog")
	// ==========================================
	http.HandleFunc("/blog", func(w http.ResponseWriter, r *http.Request) {
		posts := getProcessedPosts() // 분리 작업이 완료된 전체 데이터 호출

		var uniqueCategories []string
		categoryMap := make(map[string]bool)
		var filteredPosts []Post
		selectedCategory := r.URL.Query().Get("category")

		for _, p := range posts {
			// 고유 카테고리 추출 (All 버튼 외의 버튼들을 만들기 위함)
			if p.Category != "" && !categoryMap[p.Category] {
				categoryMap[p.Category] = true
				uniqueCategories = append(uniqueCategories, p.Category)
			}

			// 카테고리를 선택하지 않았거나(""), 선택한 카테고리와 일치하는 글만 담기
			if selectedCategory == "" || p.Category == selectedCategory {
				filteredPosts = append(filteredPosts, p)
			}
		}

		// 템플릿으로 전달할 통합 데이터 구조체
		pageData := struct {
			Posts      []Post
			Categories []string
		}{
			Posts:      filteredPosts,
			Categories: uniqueCategories,
		}

		// 블로그 템플릿 파일 이름 (사용자 환경의 html 파일명과 동일한지 확인해주세요)
		tmpl, err := template.ParseFiles("list.html", "sidebar.html")
		if err != nil {
			http.Error(w, "블로그 템플릿 에러", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, pageData) // 블로그 화면에는 필터링된 글과 카테고리 목록 전달
	})

	// ==========================================
	// [라우터 3] 상세 글 렌더링 화면 ("/post/")
	// ==========================================
	http.HandleFunc("/post/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/post/")
		if id == "" {
			http.NotFound(w, r)
			return
		}

		posts := getProcessedPosts()
		var currentPost *Post
		var uniqueCategories []string
		categoryMap := make(map[string]bool)

		for i := range posts {
			if posts[i].PageID == id {
				currentPost = &posts[i]
			}
			if posts[i].Category != "" && !categoryMap[posts[i].Category] {
				categoryMap[posts[i].Category] = true
				uniqueCategories = append(uniqueCategories, posts[i].Category)
			}
		}

		if currentPost == nil {
			http.NotFound(w, r)
			return
		}

		// 🚨 핵심 수정: 사용자님의 html 코드 {{.Category}}에 완벽히 맞추기 위해 
		// Post 구조체를 최상위로 '임베딩(*Post)' 합니다.
		pageData := struct {
			*Post               // 이렇게 하면 .Post.Category가 아니라 .Category로 바로 접근 가능합니다!
			Categories []string // 사이드바가 뻗지 않도록 카테고리도 함께 전달
		}{
			Post:       currentPost,
			Categories: uniqueCategories,
		}

		tmpl, err := template.ParseFiles("post.html", "sidebar.html")
		if err != nil {
			http.Error(w, "상세 페이지 템플릿 로드 에러", http.StatusInternalServerError)
			return
		}
		
		if err := tmpl.Execute(w, pageData); err != nil {
			log.Printf("🚨 템플릿 렌더링 실패 원인: %v\n", err)
		}
	})

	// ==========================================
	// [라우터 4] Tech Stack 화면 ("/stack")
	// ==========================================
	http.HandleFunc("/stack", func(w http.ResponseWriter, r *http.Request) {
		posts := getProcessedPosts()
		var uniqueCategories []string
		categoryMap := make(map[string]bool)

		// 사이드바를 그리기 위해 카테고리 목록만 추출합니다.
		for i := range posts {
			if posts[i].Category != "" && !categoryMap[posts[i].Category] {
				categoryMap[posts[i].Category] = true
				uniqueCategories = append(uniqueCategories, posts[i].Category)
			}
		}

		// 사이드바가 다운되지 않도록 Categories 데이터를 담아서 넘깁니다.
		pageData := struct {
			Categories []string
		}{
			Categories: uniqueCategories,
		}

		tmpl, err := template.ParseFiles("stack.html", "sidebar.html")
		if err != nil {
			log.Println("🚨 Stack 페이지 템플릿 로드 에러:", err)
			http.Error(w, "템플릿 에러", http.StatusInternalServerError)
			return
		}
		
		if err := tmpl.Execute(w, pageData); err != nil {
			log.Println("🚨 Stack 페이지 렌더링 에러:", err)
		}
	})

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// 서버 구동 (포트는 도커 환경에 맞게 조정하세요)
	log.Println("Go 서버 실행 중...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}