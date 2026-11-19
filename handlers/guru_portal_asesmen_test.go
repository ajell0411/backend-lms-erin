package handlers

import "testing"

func TestGuruValidasiSoalMendukungEsai(t *testing.T) {
	input := guruSoalInput{Teks: "Jelaskan prosesnya", Jenis: "esai", Poin: 4}
	if message := guruValidasiSoal(&input); message != "" {
		t.Fatalf("valid essay rejected: %s", message)
	}
	if input.Jenis != "esai" || input.Kunci != "" || len(input.Opsi) != 0 {
		t.Fatalf("essay options and key should be empty, got %#v", input)
	}
}

func TestGuruHitungNilaiMenungguPenilaianEsai(t *testing.T) {
	questions := []guruSoal{
		{ID: 1, Nomor: 1, Jenis: "pilihan_ganda", Kunci: "B", Poin: 2},
		{ID: 2, Nomor: 2, Jenis: "esai", Poin: 3},
	}
	answers := map[int]string{1: "b"}

	score, correct, pending := guruHitungNilaiAsesmen(questions, answers, map[int]int{})
	if score != nil || correct != 1 || !pending {
		t.Fatalf("expected pending essay with one correct answer, got score=%v correct=%d pending=%v", score, correct, pending)
	}

	score, correct, pending = guruHitungNilaiAsesmen(questions, answers, map[int]int{2: 2})
	if score == nil || *score != 80 || correct != 1 || pending {
		t.Fatalf("expected final weighted score 80, got score=%v correct=%d pending=%v", score, correct, pending)
	}
}
