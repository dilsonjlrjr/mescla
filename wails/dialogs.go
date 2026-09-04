package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	"github.com/sqweek/dialog"
)

type DialogService struct{}

func NewDialogService() *DialogService {
	return &DialogService{}
}

func (d *DialogService) SaveFile(title string, filterName string, filterPattern string) (string, error) {
	return dialog.File().Title(title).
		Filter(filterName, filterPattern).
		Save()
}

func (d *DialogService) OpenFile(title string, filterName string, filterPattern string) (string, error) {
	return dialog.File().Title(title).
		Filter(filterName, filterPattern).
		Load()
}

func (d *DialogService) SavePNG() (string, error) {
	return d.SaveFile("Salvar imagem PNG", "Imagem PNG", "*.png")
}

func (d *DialogService) SaveJSON() (string, error) {
	return d.SaveFile("Salvar documento JSON", "Documento JSON", "*.json")
}

func (d *DialogService) SaveHTML() (string, error) {
	return d.SaveFile("Salvar documento HTML", "Documento HTML", "*.html")
}

func (d *DialogService) OpenPlan() (string, error) {
	return d.OpenFile("Abrir plano de pintura", "Plano Mescla", "*.mesclaplan")
}

func (d *DialogService) SaveFileWithData(filePath string, dataBase64 string, ext string) error {
	if ext != "" && !strings.HasSuffix(filePath, ext) {
		filePath = filePath + ext
	}

	decoded, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return err
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, decoded, 0644)
}