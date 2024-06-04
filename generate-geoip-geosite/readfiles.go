package main

import (
	"bufio"
	"errors"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FileData структура для хранения информации о файле
type FileData struct {
	Path        string           // полный путь к файлу
	IsInclude   bool             // true, если файл "include" и false, если файл "exclude"
	IsIP        bool             // true, если файл c IP-адресами и false, если файл с доменами
	IsRegexp    bool             // true, если файл с регулярными выражениями
	Category    string           // категория файла
	Content     []string         // содержимое файла
	Regex       []*regexp.Regexp // скомпилированные Regex выражения из файла
	IpAddresses []net.IP         // содержимое файла (ip-адреса)
	IpNetworks  []net.IPNet      // содержимое файла (ip сети)
	// ExcludeData []string // содержимое файла exclude с регулярными выражениями
}

func processFiles(folderPath string) ([]FileData, error) {
	// Получаем список файлов в папке
	files, err := getFilesInFolder(folderPath)
	if err != nil {
		return nil, err
	}

	// Создаем массив структур FileData
	fileDataArray := make([]FileData, 0)

	// Обрабатываем каждый файл и заполняем массив структур
	for _, file := range files {
		fileData, err := getFileInfo(file)
		if err != nil {
			logWarn.Printf("file '%s' skipped: %v", file, err)
			continue
		}
		logInfo.Printf("file '%s' successfully read", file)

		fileDataArray = append(fileDataArray, *fileData)
	}

	return fileDataArray, nil
}

// getFilesInFolder возвращает список .lst и .rgx файлов в заданной папке
func getFilesInFolder(folderPath string) ([]string, error) {
	var files []string
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// getFileInfo по названию файла определяет параметры файла и читает его, возвращает структуру с данными и содержимым
func getFileInfo(filePath string) (*FileData, error) {
	// Получаем имя файла без пути к нему и убираем расширение
	fileName := filepath.Base(filePath)
	fileExtension := filepath.Ext(fileName)
	fileNameWithoutExt := strings.TrimSuffix(fileName, fileExtension)

	// Проверяем расширение на .lst и .rgx
	if fileExtension != ".lst" && fileExtension != ".rgx" {
		return nil, errors.New("'" + fileExtension + "' is invalid extension, expected '.lst' or '.rgx'")
	}

	// Разделяем имя файла получая 3 значения
	parts := strings.Split(fileNameWithoutExt, "-")
	if len(parts) < 3 {
		return nil, errors.New("expected at least 3 values in the file name: include/exclude, ip/domain, category_name")
	}

	// Определяем файл типа include или exclude
	include, err := checkIncludeExclude(parts[0])
	if err != nil {
		return nil, err
	}

	// Определяем файл с IP-адресами или доменами
	ip, err := checkIpDomain(parts[1])
	if err != nil {
		return nil, err
	}
	// Считываем категорию
	category := parts[2]

	if fileExtension == ".rgx" {
		// Если файл с регулярками, получаем массив скомпилированных регулярных выражений
		content, err := readRegexFile(filePath)
		if err != nil {
			return nil, err
		}
		return &FileData{
			Path:      filePath,
			IsInclude: include,
			IsIP:      ip,
			IsRegexp:  true,
			Category:  category,
			Regex:     content,
		}, nil
	} else {
		// Если обычный, получаем массив строк
		content, err := readFile(filePath)
		if err != nil {
			return nil, err
		}
		var ipNetworks []net.IPNet
		var ipAddresses []net.IP
		// Если список с IP адресами, то парсим их
		if ip {
			ipNetworks, ipAddresses = parseIPsAndNetworks(content)
			logInfo.Println("parsed", len(ipAddresses), "IP addresses")
			logInfo.Println("parsed", len(ipNetworks), "IP networks")
		}
		return &FileData{
			Path:        filePath,
			IsInclude:   include,
			IsIP:        ip,
			IsRegexp:    false,
			Category:    category,
			Content:     content,
			IpNetworks:  ipNetworks,
			IpAddresses: ipAddresses,
		}, nil
	}
}

func parseIPsAndNetworks(Content []string) ([]net.IPNet, []net.IP) {

	var networks []net.IPNet
	var ips []net.IP

	for _, address := range Content {
		_, ipNet, err := net.ParseCIDR(address)
		if err == nil {
			ones, bits := ipNet.Mask.Size()
			if ones == bits { // Если маска равна длине адреса, это одиночный хост
				ips = append(ips, ipNet.IP)
				// fmt.Print(ipNet, ones, bits, "\n")
				continue
			}
			networks = append(networks, *ipNet)
		} else {
			ip := net.ParseIP(address)
			if ip != nil {
				ips = append(ips, ip)
			} else {
				logWarn.Printf("invalid IP address or subnet: %s", address)
			}
		}
	}
	return networks, ips
}

// checkIncludeExclude проверяет входную строку на include и exclude
func checkIncludeExclude(input string) (bool, error) {
	lowerInput := strings.ToLower(input)

	switch lowerInput {
	case "include":
		return true, nil
	case "exclude":
		return false, nil
	default:
		return false, errors.New("invalid value, expected 'include' or 'exclude'")
	}
}

// checkIpDomain проверяет входную строку на ip и domain
func checkIpDomain(input string) (bool, error) {
	lowerInput := strings.ToLower(input)

	switch lowerInput {
	case "ip":
		return true, nil
	case "domain":
		return false, nil
	default:
		return false, errors.New("invalid value, expected 'ip' or 'domain'")
	}
}

// readFile читает строки из файла
func readFile(filePath string) ([]string, error) {
	// Читаем файл filePath
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Переменная со строками для результата
	var content []string

	// Запускаем чтение файла построчно
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Пропускаем комментарии
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Пропускаем пустые строки
		if len(line) == 0 {
			continue
		}
		// Добавляем в результирующую переменную
		content = append(content, line)

	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return content, nil
}

// readFile читает строки из файла
func readRegexFile(filePath string) ([]*regexp.Regexp, error) {
	// Читаем файл filePath
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Переменная со строками для результата
	var regex []*regexp.Regexp

	// Запускаем чтение файла построчно
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Пропускаем комментарии
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Пропускаем пустые строки
		if len(line) == 0 {
			continue
		}

		// Компилируем полученную регулярку
		rx, err := regexp.Compile(line)
		if err != nil {
			logWarn.Println(err)
			continue
		}

		// Если удачно, добавляем регулярку в исключающий массив
		regex = append(regex, rx)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return regex, nil
}
