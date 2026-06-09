# 🚀 Hybrid-Backend Web Hosting & Automated Sync Pipeline

> **Notion API, Python, Go, Docker, Cloudflare를 활용한 Zero-Trust 정적 웹 서버 구축 및 데이터 자동화 파이프라인**

---

## 📌 Project Overview
본 프로젝트는 단순한 포트폴리오 홍보용 정적 웹 사이트 서빙을 넘어, **인프라 비용 최적화, 보안성 확보, 그리고 이기종 언어(Python & Go) 간의 데이터 동기화 자동화 파이프라인 구축**에 초점을 맞춘 백엔드 및 인프라 엔지니어링 프로젝트입니다.

외부 클라우드 호스팅 서비스에 의존하지 않고, 온프레미스(로컬 환경)에서 Docker 컨테이너 기반으로 웹 서버를 독립적으로 격리·구동하며, 안정적인 데이터 패치와 실시간 업데이트 흐름을 아키텍처적으로 구현했습니다.

---

## 🛠 Tech Stack & Architecture

### 1. Infra & Deployment
* **Docker:** 웹 애플리케이션 및 동기화 스크립트를 독립된 컨테이너 환경으로 가상화하여 환경 의존성 제거 및 배포 안정성 확보.
* **Cloudflare Tunnel (`cloudflared`):** 전통적인 포트 포워딩 방식의 보안 취약점을 해결하기 위한 **Zero-Trust 아웃바운드 터널링** 채택. 외부 인바운드 포트를 완전 은닉하여 로컬 인프라 보안 극대화.

### 2. Data Pipeline & Automation
* **Blog Auto-Loader (Python + Notion API):** 노션 데이터베이스를 CMS로 활용. Python 스크립트가 Notion API를 통해 최신 블로그 콘텐츠를 주기적으로 패치(Fetch)하고 구조화된 JSON 데이터로 변환하는 파이프라인 자동화.
* **HTML Real-time Updater (Go):** 생성된 JSON 데이터를 파싱하여 정적 HTML 에셋을 상시 업데이트하고 동적으로 반영하는 경량·고성능 동기화 코어 엔진을 Go 언어로 구현.

---

## 🏗 System Architecture Diagram

[ Notion CMS ]

│ (Notion API)

▼

[ Python Sync Script ] ──(Extract & Transform)──> [ data.json ]

│

▼

[ Go Update Engine ]

│ (Generate / Update)

▼

[ Static HTML Assets ]

│

▼

[ Docker / Nginx ]

▲

│ (Secure Outbound Tunnel)

[ Cloudflare Tunnel ]

▲

│ (HTTPS)

[ User Web ]

---

## 🎯 Key Engineering Points (핵심 성과)

* **보안 중심의 네트워크 설계:** 외부 공격 표면(Attack Surface)을 최소화하기 위해 공인 IP 노출 없이 Cloudflare Tunnel을 통해 안전한 SSL/TLS 암호화 웹 서빙 통로를 개설했습니다.
* **이기종 언어의 적재적소 활용 (Polyglot):** * 풍부한 생태계를 가진 **Python**을 사용하여 다양한 외부 API(Notion) 연동 및 JSON 데이터 변환 로직을 신속하게 안정화했습니다.
  * 컴파일 언어인 **Go**의 빠른 바이너리 실행 속도와 가벼운 리소스 점유율을 활용하여, I/O 작업(JSON 파싱 및 HTML 파일 쓰기)의 병목을 최소화하고 상시 업데이트 성능을 극대화했습니다.
* **WIP (Work In Progress):** 현재 로컬 가상화 환경에서의 아키텍처 검증을 완료했으며, 향후 무중단 배포 및 모니터링 환경으로의 확장을 준비 중입니다.

---

## 💾 Directory Structure
```
r3lis-site
├── .gitignore
├── LICENSE
├── README.md
├── docker-compose.yml
├── /data         # Notion API 통해 추출된 데이터 저장 위치
│   └── r3lis-posts.json
├── /python       # Notion API 데이터 추출 및 JSON 변환 모듈
│   ├── .env              # (보안 요소 저장, 리포지토리 내 미포함)
│   ├── Dockerfile
│   ├── requirements.txt  # 사용 시 필요 모듈 기록
│   └── fetch_notion_data.py
├── /nginx
│   └── nginx.conf
└── /go           # JSON 기반 HTML 실시간 제어 및 동기화 엔진
    ├── /static           # 로고 이미지 및 기본 테마 CSS 저장
    │   ├── 21_20240831124804.png
    │   └── style.css
    ├── Dockerfile
    ├── go.mode
    ├── main.go
    ├── index.html
    ├── list.html
    ├── post.html
    ├── sidebar.html
    └── stack.html
```

📄 License
This project is licensed under the MIT License - see the LICENSE file for details.
