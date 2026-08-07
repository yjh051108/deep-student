// 文本切片器，参考 Rust embedding_chunker.rs。
//
// 切片策略：
// 1. 按段落边界（\n\n）切
// 2. 单段落过长时按句号边界（。.！!？?）切
// 3. 相邻切片重叠 chunkOverlap 字符
// 4. 跳过空切片和小于 MinChunkSize 的尾片

package index

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ChunkText 将文本切分为多个 Chunk。
//
// opts 中的 ChunkSize/ChunkOverlap/MinChunkSize 控制切片参数。
// 返回的 Chunk 仅填充 Pos/Content/TokenCount，URI/ID/CreatedAt 由调用方补充。
func ChunkText(text string, opts IndexOptions) []Chunk {
	if opts.ChunkSize <= 0 {
		opts = DefaultOptions()
	}
	if opts.ChunkOverlap < 0 {
		opts.ChunkOverlap = 0
	}
	if opts.MinChunkSize < 0 {
		opts.MinChunkSize = 0
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// 文本不超过 chunkSize，直接返回单块
	if utf8.RuneCountInString(text) <= opts.ChunkSize {
		tc := estimateTokens(text)
		if tc == 0 {
			tc = 1
		}
		return []Chunk{{
			Pos:        0,
			Content:    text,
			TokenCount: tc,
		}}
	}

	// 按段落分割
	paragraphs := splitParagraphs(text)

	// 将段落组装成块
	var rawChunks []rawChunk
	current := strings.Builder{}
	currentPos := 0 // 当前块在原文中的字符偏移

	for _, para := range paragraphs {
		paraText := para.text
		paraRunes := utf8.RuneCountInString(paraText)

		// 单段落超过 chunkSize，需要按句子分割
		if paraRunes > opts.ChunkSize {
			// 先保存当前块
			if current.Len() > 0 {
				rawChunks = append(rawChunks, rawChunk{
					text: current.String(),
					pos:  currentPos,
				})
				current.Reset()
			}

			// 按句子分割段落
			sentences := splitSentences(paraText)
			for _, sent := range sentences {
				sentRunes := utf8.RuneCountInString(sent)
				if sentRunes > opts.ChunkSize {
					// 单句仍超长，硬切割
					hardChunks := hardChunk(sent, opts.ChunkSize)
					for _, hc := range hardChunks {
						rawChunks = append(rawChunks, rawChunk{text: hc, pos: 0})
					}
					continue
				}

				// 尝试加入当前块
				if current.Len()+sentRunes+1 > opts.ChunkSize && current.Len() > 0 {
					rawChunks = append(rawChunks, rawChunk{
						text: current.String(),
						pos:  currentPos,
					})
					current.Reset()
				}
				if current.Len() > 0 {
					current.WriteString(" ")
				}
				if current.Len() == 0 {
					currentPos = para.start
				}
				current.WriteString(sent)
			}
			continue
		}

		// 加入当前块会超限
		if current.Len()+paraRunes+2 > opts.ChunkSize && current.Len() > 0 {
			rawChunks = append(rawChunks, rawChunk{
				text: current.String(),
				pos:  currentPos,
			})
			current.Reset()
			currentPos = para.start
			current.WriteString(paraText)
		} else {
			if current.Len() == 0 {
				currentPos = para.start
			}
			if current.Len() > 0 {
				current.WriteString("\n\n")
			}
			current.WriteString(paraText)
		}
	}

	// 保存最后一块
	if current.Len() > 0 {
		rawChunks = append(rawChunks, rawChunk{
			text: current.String(),
			pos:  currentPos,
		})
	}

	if len(rawChunks) == 0 {
		return nil
	}

	// 应用重叠
	if opts.ChunkOverlap > 0 && len(rawChunks) > 1 {
		rawChunks = applyOverlap(rawChunks, opts.ChunkOverlap)
	}

	// 转换为 Chunk，过滤过小的尾片
	var result []Chunk
	for i, rc := range rawChunks {
		content := strings.TrimSpace(rc.text)
		runeCount := utf8.RuneCountInString(content)

		// 跳过空切片
		if content == "" {
			continue
		}

		// 跳过小于 MinChunkSize 的尾片（非首片）
		if i > 0 && runeCount < opts.MinChunkSize {
			continue
		}

		tc := estimateTokens(content)
		if tc == 0 {
			tc = 1
		}
		result = append(result, Chunk{
			Pos:        i,
			Content:    content,
			TokenCount: tc,
		})
	}

	return result
}

// rawChunk 内部切片结构。
type rawChunk struct {
	text string
	pos  int
}

// paragraph 段落及其在原文中的起始字符偏移。
type paragraph struct {
	text  string
	start int
}

// splitParagraphs 按空行分割段落，记录每个段落的字符偏移。
func splitParagraphs(text string) []paragraph {
	var result []paragraph
	runeOffset := 0
	for _, para := range strings.Split(text, "\n\n") {
		trimmed := strings.TrimSpace(para)
		if trimmed == "" {
			runeOffset += utf8.RuneCountInString(para) + 2 // \n\n
			continue
		}
		result = append(result, paragraph{
			text:  trimmed,
			start: runeOffset,
		})
		runeOffset += utf8.RuneCountInString(para) + 2
	}
	return result
}

// splitSentences 按句号边界分割句子。
func splitSentences(text string) []string {
	sentenceEndings := map[rune]bool{
		'。': true, '！': true, '？': true,
		'.': true, '!': true, '?': true,
		'；': true, ';': true, '\n': true,
	}

	var sentences []string
	var current strings.Builder

	for _, r := range text {
		current.WriteRune(r)
		if sentenceEndings[r] {
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}

	if current.Len() > 0 {
		s := strings.TrimSpace(current.String())
		if s != "" {
			sentences = append(sentences, s)
		}
	}

	return sentences
}

// hardChunk 按字符数硬切割（UTF-8 安全）。
func hardChunk(text string, chunkSize int) []string {
	runes := []rune(text)
	var result []string

	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, string(runes[i:end]))
	}

	return result
}

// applyOverlap 为每个块（除第一个）添加前一块末尾的重叠文本。
func applyOverlap(chunks []rawChunk, overlap int) []rawChunk {
	if len(chunks) <= 1 || overlap <= 0 {
		return chunks
	}

	result := make([]rawChunk, len(chunks))
	result[0] = chunks[0]

	for i := 1; i < len(chunks); i++ {
		prev := chunks[i-1].text
		tail := tailByRunes(prev, overlap)
		tail = strings.TrimSpace(tail)
		if tail != "" {
			result[i] = rawChunk{
				text: tail + " " + chunks[i].text,
				pos:  chunks[i].pos,
			}
		} else {
			result[i] = chunks[i]
		}
	}
	return result
}

// tailByRunes 取文本末尾 n 个 rune 的子串。
func tailByRunes(text string, n int) string {
	runes := []rune(text)
	if n >= len(runes) {
		return text
	}
	return string(runes[len(runes)-n:])
}

// estimateTokens 估算文本的 token 数。
//
// 粗略估算：中文按字符数 × 1.5，英文按空格分词数 × 1.3，混合时取较大值。
func estimateTokens(text string) int {
	var chineseChars int
	var asciiWords int

	inWord := false
	for _, r := range text {
		if isCJK(r) {
			chineseChars++
			inWord = false
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				asciiWords++
				inWord = true
			}
		} else {
			inWord = false
		}
	}

	cnTokens := int(float64(chineseChars) * 1.5)
	enTokens := int(float64(asciiWords) * 1.3)

	if cnTokens > enTokens {
		return cnTokens
	}
	return enTokens
}

// isCJK 判断是否为 CJK 统一汉字。
func isCJK(r rune) bool {
	if r >= '\u4E00' && r <= '\u9FFF' {
		return true
	}
	if r >= '\u3400' && r <= '\u4DBF' {
		return true
	}
	if r >= '\uF900' && r <= '\uFAFF' {
		return true
	}
	return false
}
