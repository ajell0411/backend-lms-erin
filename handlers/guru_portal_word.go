package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
	"e-class-backend/models"
)

var (
	wordSoal          = regexp.MustCompile(`(?i)^soal(\s+\d+)?\s*:\s*(.*)$`)
	wordNomorSoal     = regexp.MustCompile(`^\d+[.)]\s*(.+)$`)
	wordOpsi          = regexp.MustCompile(`(?i)^(opsi\s*)?[\(\[]?([A-E])[\)\]]?\s*[.)\-:]?\s+(.+)$`)
	wordKunci         = regexp.MustCompile(`(?i)^(kunci|kunci jawaban|jawaban|jawaban benar|jawaban yang benar)\s*[:=\-]\s*([A-E])[\.)]?$`)
	wordKunciSingkat  = regexp.MustCompile(`(?i)^([A-E])[\.)]?$`)
	wordJudulKunci    = regexp.MustCompile(`(?i)^(kunci jawaban|daftar kunci jawaban)\s*:?\s*$`)
	wordKunciBernomor = regexp.MustCompile(`(?i)^(\d+)[.)]\s*([A-E])[\.)]?$`)
)

func teksWordDocx(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	for _, file := range zr.File {
		if file.Name != "word/document.xml" {
			continue
		}
		r, err := file.Open()
		if err != nil {
			return "", err
		}
		defer r.Close()
		decoder := xml.NewDecoder(r)
		var out strings.Builder
		inParagraph := false
		for {
			tok, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}
			switch t := tok.(type) {
			case xml.StartElement:
				if t.Name.Local == "p" {
					inParagraph = true
				}
			case xml.EndElement:
				if t.Name.Local == "p" && inParagraph {
					out.WriteByte('\n')
					inParagraph = false
				}
			case xml.CharData:
				if inParagraph {
					out.Write([]byte(t))
				}
			}
		}
		return out.String(), nil
	}
	return "", fmt.Errorf("isi dokumen Word tidak ditemukan")
}

func parseWordSoal(text string) ([]guruSoalInput, error) {
	var result []guruSoalInput
	var current *guruSoalInput
	var options []guruOpsi
	globalKeys := make(map[int]string)
	readingKeys := false
	flush := func() error {
		if current == nil {
			return nil
		}
		current.Teks = strings.TrimSpace(current.Teks)
		if current.Teks == "" {
			return fmt.Errorf("teks pertanyaan tidak boleh kosong")
		}
		current.Opsi = options
		if len(options) < 4 || len(options) > 5 {
			return fmt.Errorf("setiap soal harus memiliki empat atau lima opsi A-E")
		}
		for i := range current.Opsi {
			current.Opsi[i].Kode = string(rune('A' + i))
			current.Opsi[i].Teks = strings.TrimSpace(current.Opsi[i].Teks)
			if current.Opsi[i].Teks == "" {
				return fmt.Errorf("opsi %s pada soal '%s' kosong", current.Opsi[i].Kode, current.Teks)
			}
		}
		result = append(result, *current)
		if len(result) > 100 {
			return fmt.Errorf("maksimal 100 soal dalam satu dokumen")
		}
		current = nil
		options = nil
		return nil
	}
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r", ""), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if wordJudulKunci.MatchString(line) {
			if err := flush(); err != nil {
				return nil, err
			}
			readingKeys = true
			continue
		}
		if readingKeys {
			match := wordKunciBernomor.FindStringSubmatch(line)
			if len(match) != 3 {
				return nil, fmt.Errorf("format kunci bernomor tidak dikenali: %s", line)
			}
			nomor, err := strconv.Atoi(match[1])
			if err != nil || nomor < 1 || nomor > 100 {
				return nil, fmt.Errorf("nomor kunci jawaban tidak valid: %s", match[1])
			}
			if _, exists := globalKeys[nomor]; exists {
				return nil, fmt.Errorf("kunci jawaban nomor %d tercantum lebih dari sekali", nomor)
			}
			globalKeys[nomor] = strings.ToUpper(match[2])
			continue
		}

		if match := wordSoal.FindStringSubmatch(line); len(match) == 3 {
			if err := flush(); err != nil {
				return nil, err
			}
			current = &guruSoalInput{Teks: strings.TrimSpace(match[2]), Jenis: "pilihan_ganda", Poin: 1}
			continue
		}
		if current == nil {
			question := line
			if match := wordNomorSoal.FindStringSubmatch(line); len(match) == 2 {
				question = strings.TrimSpace(match[1])
			}
			current = &guruSoalInput{Teks: question, Jenis: "pilihan_ganda", Poin: 1}
			continue
		}

		if match := wordOpsi.FindStringSubmatch(line); len(match) == 4 {
			kode := strings.ToUpper(match[2])
			if kode != string(rune('A'+len(options))) {
				return nil, fmt.Errorf("opsi harus berurutan mulai dari A sampai D atau E")
			}
			options = append(options, guruOpsi{Kode: kode, Teks: strings.TrimSpace(match[3])})
			continue
		}

		if match := wordKunci.FindStringSubmatch(line); len(match) == 3 {
			current.Kunci = strings.ToUpper(match[2])
			continue
		}
		if match := wordKunciSingkat.FindStringSubmatch(line); len(match) == 2 && len(options) >= 4 {
			current.Kunci = strings.ToUpper(match[1])
			continue
		}
		if match := wordNomorSoal.FindStringSubmatch(line); len(match) == 2 && len(options) >= 4 {
			if err := flush(); err != nil {
				return nil, err
			}
			current = &guruSoalInput{Teks: strings.TrimSpace(match[1]), Jenis: "pilihan_ganda", Poin: 1}
			continue
		}
		if current.Kunci != "" || len(options) >= 4 {
			if err := flush(); err != nil {
				return nil, err
			}
			current = &guruSoalInput{Teks: line, Jenis: "pilihan_ganda", Poin: 1}
			continue
		}
		if len(options) == 0 {
			current.Teks = strings.TrimSpace(current.Teks + " " + line)
			continue
		}
		return nil, fmt.Errorf("format baris opsi atau kunci tidak dikenali: %s", line)
	}
	if err := flush(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("tidak ada soal yang ditemukan")
	}
	for nomor, kunci := range globalKeys {
		if nomor > len(result) {
			return nil, fmt.Errorf("kunci jawaban nomor %d tidak memiliki soal", nomor)
		}
		if result[nomor-1].Kunci != "" && result[nomor-1].Kunci != kunci {
			return nil, fmt.Errorf("kunci jawaban soal %d berbeda antara bagian soal dan daftar kunci", nomor)
		}
		result[nomor-1].Kunci = kunci
	}
	for i := range result {
		if result[i].Kunci == "" {
			return nil, fmt.Errorf("kunci jawaban soal %d belum ditemukan", i+1)
		}
		if pesan := guruValidasiSoal(&result[i]); pesan != "" {
			return nil, fmt.Errorf("soal %d: %s", i+1, pesan)
		}
	}
	return result, nil
}

func ImporWordGuruSoal(c *gin.Context) {
	guruID, ok := guruPortalID(c)
	if !ok {
		return
	}
	id, ok := guruPortalParamID(c)
	if !ok {
		return
	}
	var asesmen models.Kuis
	if err := config.DB.Where("id = ? AND guru_id = ?", id, guruID).First(&asesmen).Error; err != nil {
		guruPortalGagal(c, http.StatusNotFound, "Asesmen tidak ditemukan")
		return
	}
	header, err := c.FormFile("file")
	if err != nil || !strings.EqualFold(filepath.Ext(header.Filename), ".docx") || header.Size > 10*1024*1024 {
		guruPortalGagal(c, http.StatusBadRequest, "Unggah file Word .docx maksimal 10 MB")
		return
	}
	file, err := header.Open()
	if err != nil {
		guruPortalGagal(c, http.StatusBadRequest, "File Word tidak dapat dibuka")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, 10*1024*1024+1))
	if err != nil || len(content) > 10*1024*1024 {
		guruPortalGagal(c, http.StatusBadRequest, "File Word tidak dapat dibaca atau terlalu besar")
		return
	}
	text, err := teksWordDocx(content)
	if err != nil {
		guruPortalGagal(c, http.StatusBadRequest, "Dokumen .docx tidak valid")
		return
	}
	imports, err := parseWordSoal(text)
	if err != nil {
		guruPortalGagal(c, http.StatusBadRequest, "Format soal Word tidak sesuai: "+err.Error())
		return
	}
	soal := guruBacaSoal(asesmen.Soal)
	for _, input := range imports {
		idBaru := 1
		for _, old := range soal {
			if old.ID >= idBaru {
				idBaru = old.ID + 1
			}
		}
		soal = append(soal, guruSoal{ID: idBaru, Teks: input.Teks, Jenis: "pilihan_ganda", Opsi: input.Opsi, Kunci: input.Kunci, Poin: 1})
	}
	if err := guruSimpanSoal(asesmen.ID, soal); err != nil {
		guruPortalGagal(c, http.StatusInternalServerError, "Soal gagal disimpan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": soal})
}
