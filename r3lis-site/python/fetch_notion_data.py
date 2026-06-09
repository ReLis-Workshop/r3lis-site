import os
import json
import asyncio
import httpx
import re        # 파이썬 기본 내장 (정규식)
import uuid      # 파이썬 기본 내장 (고유 파일명 생성)
from dotenv import load_dotenv

IMAGE_SAVE_DIR = "/data/images"
os.makedirs(IMAGE_SAVE_DIR, exist_ok=True)

# 1. 환경 변수 안전하게 로드
load_dotenv()
NOTION_API_KEY = os.getenv("NOTION_API_KEY", "").strip()
NOTION_DATABASE_ID = os.getenv("NOTION_DATABASE_ID", "").strip()

def parse_rich_text(rich_text_array: list) -> str:
    """텍스트의 디자인(굵기, 기울임 등)을 마크다운으로 변환"""
    parsed_text = ""
    for t in rich_text_array:
        span = t.get("plain_text", "")
        if not span:
            continue
        
        annotations = t.get("annotations", {})
        if annotations.get("code"): span = f"`{span}`"
        if annotations.get("bold"): span = f"**{span}**"
        if annotations.get("italic"): span = f"*{span}*"
        if annotations.get("strikethrough"): span = f"~~{span}~~"
            
        href = t.get("href")
        if href: span = f"[{span}]({href})"
        parsed_text += span
        
    return parsed_text

async def process_markdown_images(client: httpx.AsyncClient, markdown_content: str) -> str:
    """본문의 노션 S3 이미지를 기존 httpx 세션으로 다운받고 경로를 치환"""
    # 마크다운 이미지 정규식: ![alt](url)
    pattern = re.compile(r'!\[([^\]]*)\]\((https?://[^\)]+)\)')
    matches = pattern.findall(markdown_content)
    
    for alt_text, img_url in matches:
        try:
            # 기존에 사용 중인 httpx 클라이언트로 이미지 다운로드
            response = await client.get(img_url, timeout=10.0)
            response.raise_for_status()
            
            file_name = f"{uuid.uuid4().hex}.png"
            file_path = os.path.join(IMAGE_SAVE_DIR, file_name)
            
            with open(file_path, 'wb') as f:
                f.write(response.content)
            
            # 서버 내부 경로로 텍스트 치환
            new_url = f"/images/{file_name}"
            markdown_content = markdown_content.replace(f"![{alt_text}]({img_url})", f"![{alt_text}]({new_url})")
            
        except Exception as e:
            print(f"이미지 다운로드 실패 ({img_url}): {e}")
            
    return markdown_content

async def fetch_page_content(client: httpx.AsyncClient, page_id: str, headers: dict) -> str:

    """개별 페이지(글)의 블록들을 가져와 본문 마크다운으로 변환"""
    url = f"https://api.notion.com/v1/blocks/{page_id}/children"
    response = await client.get(url, headers=headers)
    
    if response.status_code != 200:
        return ""
    
    blocks = response.json().get("results", [])
    content_lines = []

    for block in blocks:
        b_type = block.get("type")
        if not b_type or b_type not in block:
            continue
            
        block_data = block[b_type]
        rich_text_array = block_data.get("rich_text", [])
        text = parse_rich_text(rich_text_array)
        
        if b_type == "heading_1": content_lines.append(f"# {text}")
        elif b_type == "heading_2": content_lines.append(f"## {text}")
        elif b_type == "heading_3": content_lines.append(f"### {text}")
        elif b_type == "bulleted_list_item": content_lines.append(f"- {text}")
        elif b_type == "numbered_list_item": content_lines.append(f"1. {text}")
        elif b_type == "quote": content_lines.append(f"> {text}")
        elif b_type == "code":
            language = block_data.get("language", "")
            raw_code = "".join([t.get("plain_text", "") for t in rich_text_array])
            content_lines.append(f"```{language}\n{raw_code}\n```")
        elif b_type == "image":
            image_type = block_data.get("type")
            image_url = block_data.get(image_type, {}).get("url", "")
            caption_array = block_data.get("caption", [])
            caption = parse_rich_text(caption_array) if caption_array else "이미지"
            if image_url: content_lines.append(f"![{caption}]({image_url})")
        elif text:
            content_lines.append(text)

    markdown_content = await process_markdown_images(client, markdown_content)

    return "\n\n".join(content_lines)

async def poll_notion_data():
    """노션 DB 목록을 긁어오고, 각 목록의 본문 수집을 지시하는 메인 함수"""
    if not NOTION_API_KEY or not NOTION_DATABASE_ID:
        print("오류: API 키 또는 데이터베이스 ID가 비어있습니다.")
        return

    url = f"https://api.notion.com/v1/databases/{NOTION_DATABASE_ID}/query"
    headers = {
        "Authorization": f"Bearer {NOTION_API_KEY}",
        "Notion-Version": "2022-06-28",
        "Content-Type": "application/json"
    }
    
    # ★ 변경 포인트 1: 시스템 생성일(timestamp)이 아닌, 사용자가 만든 "작성일" 속성(property) 기준으로 정렬합니다.
    payload = {"sorts": [{"property": "작성일", "direction": "descending"}]}

    print("노션 DB 글 목록을 로드하는 중...")

    try:
        async with httpx.AsyncClient() as client:
            response = await client.post(url, headers=headers, json=payload)

            if response.status_code != 200:
                print(f"API 요청 실패: {response.status_code}")
                return

            results = response.json().get("results", [])
            post_data_list = []

            print(f"총 {len(results)}개의 글 본문 파싱을 시작합니다...")

            for page in results:
                page_id = page.get("id")
                props = page.get("properties", {})
                title = "제목 없음"
                
                # 제목 추출
                for prop_name, prop_data in props.items():
                    if prop_data.get("type") == "title":
                        title_array = prop_data.get("title", [])
                        if title_array: title = "".join([t.get("plain_text", "") for t in title_array])
                        break
                
                # ★ 변경 포인트 2: "작성일" 속성에서 지정된 날짜 가져오기 (비어있으면 시스템 생성일로 대체)
                custom_date = ""
                date_prop = props.get("작성일", {})
                if date_prop.get("type") == "date" and date_prop.get("date"):
                    custom_date = date_prop["date"].get("start")
                
                # 만약 노션에서 '작성일'을 실수로 지정하지 않았다면 기본 생성일로 폴백(Fallback) 처리
                if not custom_date:
                    custom_date = page.get("created_time")
                
                print(f" -> '{title}' 본문 파싱 중...")
                
                content = await fetch_page_content(client, page_id, headers)

                post_data_list.append({
                    "page_id": page_id,
                    "title": title if title else "제목 없음",
                    "created_at": custom_date,  # 사용자가 지정한 '작성일'이 우선적으로 들어갑니다.
                    "last_edited": page.get("last_edited_time"),
                    "url": page.get("url"),
                    "content": content
                })

            # 도커 공유 볼륨 경로에 저장
            output_filename = "data/r3lis_posts.json"
            with open(output_filename, "w", encoding="utf-8") as f:
                json.dump(post_data_list, f, ensure_ascii=False, indent=4)

            print(f"\n성공: 모든 글과 본문 파싱이 완료되어 '{output_filename}'에 저장되었습니다.")

    except Exception as e:
        print(f"치명적 오류 발생: {e}")

async def main_loop():
    """5분마다 위 과정을 반복하는 스케줄러"""
    while True:
        print("\n========================================")
        print("▶ [스케줄러] 노션 데이터 동기화를 시작합니다.")
        print("========================================")
        
        await poll_notion_data()
        
        print("\n[스케줄러] 동기화 완료. 다음 업데이트까지 5분(300초) 대기합니다...")
        await asyncio.sleep(300)

if __name__ == "__main__":
    try:
        asyncio.run(main_loop())
    except KeyboardInterrupt:
        print("\n[스케줄러] 사용자에 의해 데이터 수집 프로그램이 종료되었습니다.")