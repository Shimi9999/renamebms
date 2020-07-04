package bmsloader

import (
  "fmt"
  "os"
  "bufio"
  "strings"
  "strconv"
  "regexp"
  "io/ioutil"
  "path/filepath"
  "encoding/json"
  "unicode/utf8"
  "crypto/sha256"
  "crypto/md5"

  "golang.org/x/text/encoding/japanese"
  "golang.org/x/text/transform"

  "../bmsobject"
)

func IsBmsPath(path string) bool {
  ext := filepath.Ext(path)
  bmsExts := []string{".bms", ".bme", ".bml", ".pms", ".bmson"}
  for _, be := range bmsExts {
    if strings.ToLower(ext) == be {
      return true
    }
  }
  return false
}

func IsBmsonPath(path string) bool {
  if filepath.Ext(path) == ".bmson" {
    return true
  }
  return false
}

func LoadBms(path string) (bmsobject.BmsFile, bool/*noobj*/, error) {
  file, err := os.Open(path)
  if err != nil {
    return bmsobject.NewBmsFile(), true, fmt.Errorf("BMSfile open error : " + err.Error())
  }
  defer file.Close()

  const (
    initialBufSize = 10000
    maxBufSize = 1000000
  )
  scanner := bufio.NewScanner(file)
  buf := make([]byte, initialBufSize)
  scanner.Buffer(buf, maxBufSize)

  bmsfile := bmsobject.NewBmsFile()
  bmsfile.Path = path
  chmap := map[string]bool{"7k": false, "10k": false, "14k": false}
  noobj := true
  for scanner.Scan() {
    line, _, err := transform.String(japanese.ShiftJIS.NewDecoder(), scanner.Text())
    if err != nil {
      return bmsobject.NewBmsFile(), true, fmt.Errorf("ShiftJIS decode error: " + err.Error())
    }

    if strings.HasPrefix(line, "#TITLE") {
      bmsfile.Title = strings.Trim(line[6:], " ")
    } else if strings.HasPrefix(line, "#SUBTITLE") {
      bmsfile.Subtitle = strings.Trim(line[9:], " ")
    } else if strings.HasPrefix(line, "#PLAYLEVEL") {
      bmsfile.Playlevel = strings.Trim(line[10:], " ")
    } else if strings.HasPrefix(line, "#DIFFICULTY") {
      bmsfile.Difficulty = strings.Trim(line[11:], " ")
    } else if strings.HasPrefix(line, "#ARTIST") {
      bmsfile.Artist = strings.Trim(line[7:], " ")
    } else if strings.HasPrefix(line, "#GENRE") {
      bmsfile.Genre = strings.Trim(line[6:], " ")
    } else if regexp.MustCompile(`#[0-9]{5}:.+`).MatchString(line) {
      chint, _ := strconv.Atoi(line[4:6])
      if (chint >= 18 && chint <= 19) || (chint >= 38 && chint <= 39) {
        chmap["7k"] = true
      } else if (chint >= 21 && chint <= 26) || (chint >= 41 && chint <= 46) {
        chmap["10k"] = true
      } else if (chint >= 28 && chint <= 29) || (chint >= 48 && chint <= 49) {
        chmap["14k"] = true
      }

      if noobj {
        if (chint >= 11 && chint <= 19) || (chint >= 21 && chint <= 29) {
          noobj = false
        }
      }
    }
  }
  if scanner.Err() != nil {
    return bmsobject.NewBmsFile(), true, fmt.Errorf("BMSfile scan error: " + scanner.Err().Error())
  }

  if filepath.Ext(path) == ".pms" {
    bmsfile.Keymode = 9
  } else if chmap["10k"] || chmap["14k"] {
    if chmap["7k"] || chmap["14k"] {
      bmsfile.Keymode = 14
    } else {
      bmsfile.Keymode = 10
    }
  } else if chmap["7k"] {
    bmsfile.Keymode = 7
  } else {
    bmsfile.Keymode = 5
  }

  bmsfile.Md5, bmsfile.Sha256, err = getBmsHash(path)
  if err != nil {
    return bmsobject.NewBmsFile(), true, fmt.Errorf("Get bmshash error: " + err.Error())
  }

  return bmsfile, noobj, nil
}

func LoadBmson(path string) (bmsobject.BmsFile, error) {
  bytes, err := ioutil.ReadFile(path)
  if err != nil {
    return bmsobject.NewBmsFile(), fmt.Errorf("BMSONfile open error : " + err.Error())
  }

  var bmson bmsobject.Bmson
  if err := json.Unmarshal(bytes, &bmson); err != nil {
    return bmsobject.NewBmsFile(), fmt.Errorf("BMSONfile unmarshal error : " + err.Error())
  }

  bmsfile := bmsobject.NewBmsFile()
  bmsfile.Path = path
  bmsfile.Title = bmson.Bmsoninfo.Title + bmson.Bmsoninfo.Subtitle
  bmsfile.Subtitle = bmson.Bmsoninfo.Chartname
  bmsfile.Playlevel = strconv.Itoa(bmson.Bmsoninfo.Level)
  bmsfile.Artist = bmson.Bmsoninfo.Artist
  bmsfile.Genre = bmson.Bmsoninfo.Genre

  keymap := map[string]int{"5k": 5, "7k": 7, "9k": 9, "10k": 10, "14k": 14, "24k": 24, "48k": 48}
  for key, value := range keymap {
    if strings.Contains(bmson.Bmsoninfo.Modehint, key) {
      bmsfile.Keymode = value
      break
    }
  }

  _, bmsfile.Sha256, err = getBmsHash(path)
  if err != nil {
    return bmsobject.NewBmsFile(), fmt.Errorf("Get bmshash error: " + err.Error())
  }

  return bmsfile, nil
}

func getBmsHash(path string) (string, string, error) {
  bmsStr, err := loadBmsFileString(path)
  if err != nil {
    return "", "", err
  }
  md5 := fmt.Sprintf("%x", md5.Sum([]byte(bmsStr)))
  sha256 := fmt.Sprintf("%x", sha256.Sum256([]byte(bmsStr)))

  return md5, sha256, nil
}

func loadBmsFileString(path string) (string, error) {
  file, err := os.Open(path)
  if err != nil {
    return "", fmt.Errorf("BMS open error : " + err.Error())
  }
  defer file.Close()

  var str string
  buf := make([]byte, 1024)
  for {
    n, err := file.Read(buf)
    if n == 0 {
      break
    }
    if err != nil {
      return "", fmt.Errorf("BMS read error : " + err.Error())
    }

    str += string(buf[:n])
  }
  return str, nil
}

func getDifficultyFromTitle(bms bmsobject.BmsFile) string {
  difficulties := []string{"beginner", "normal", "hyper", "another", "insane"}
  difnums := []string{"1", "2", "3", "4", "5"}
  brackets := [][]string{{`\[`,`\]`}, {`\(`,`\)`}, {"-","-"}, {`【`,`】`}}

  fulltitle := strings.ToLower(strings.TrimSpace(bms.Title + bms.Subtitle))
  // first match black another(=insane)
  for _, bracket := range brackets {
    s := ".+" + bracket[0] + ".*black.*another.*" + bracket[1] + "$"
    if regexp.MustCompile(s).MatchString(fulltitle) {
      return difnums[4]
    }
  }

  for index, difficulty := range difficulties {
    for _, bracket := range brackets {
      s := ".+" + bracket[0] + ".*" + difficulty + ".*" + bracket[1] + "$"
      if regexp.MustCompile(s).MatchString(fulltitle) {
        return difnums[index]
      }
    }
  }
  return ""
}

func getPlainName(path string) string {
  return filepath.Base(path[:len(path) - len(filepath.Ext(path))])
}

func getDifficultyFromPlainName(plainname string, justmatch bool) string {
  if plainname == "" {
    return ""
  }
  difficulties := []string{}
  predifs := []string{"", "sp", "dp", "5", "7", "9", "14", "5k", "7k", "9k", "14k"}
  difs := []string{"b", "n", "h", "a", "i", "beginner", "normal", "hyper", "another", "insane"}
  for _, predif := range predifs {
    for _, dif := range difs {
      difficulties = append(difficulties, predif + dif)
    }
  }
  pres := []string{"", " ", "-", "_"}
  if (justmatch) {
    pres = []string{""}
  }
  brackets := [][]string{{`\[`,`\]`}, {`\(`,`\)`}}
  difnums := []string{"1", "2", "3", "4", "5"}

  plainname = strings.ToLower(plainname)
  for index, difficulty := range difficulties {
    for _, pre := range pres {
      if pre == "" {
        if plainname == difficulty {
          return difnums[index % 5]
        }
      } else if strings.HasSuffix(plainname, pre + difficulty) {
        return difnums[index % 5]
      }
    }
    for _, bracket := range brackets {
      s := ".+" + bracket[0] + difficulty + bracket[1] + "$"
      if regexp.MustCompile(s).MatchString(plainname) {
        return difnums[index % 5]
      }
    }
  }
  return ""
}

func getDifficultyFromPath(bms bmsobject.BmsFile) string {
  return getDifficultyFromPlainName(getPlainName(bms.Path), false)
}

// 複数のbmsファイル名から先頭一致した文字列を排除してDIFFICULTYマッチを行う
func findDifficultyFromDirectory(bmsdir *bmsobject.BmsDirectory) {
  if len(bmsdir.Bmsfiles) < 2 {
    return
  }

  plainnames := []string{}
  for _, bmsfile := range bmsdir.Bmsfiles {
    plainnames = append(plainnames, strings.ToLower(getPlainName(bmsfile.Path)))
  }

  var matchstr string
  i := 1
  for ; i <= utf8.RuneCountInString(plainnames[0]); i++ {
    for j := 1; j < len(plainnames); j++ {
      if utf8.RuneCountInString(plainnames[j]) < i || plainnames[0][:i] != plainnames[j][:i] {
        if i == 1 {
          return
        } else {
          goto OUT
        }
      }
    }
  }
  OUT:
  matchstr = plainnames[0][:i-1]
  for index, _ := range bmsdir.Bmsfiles {
    bmsdir.Bmsfiles[index].Difficulty = getDifficultyFromPlainName(plainnames[index][utf8.RuneCountInString(matchstr):], true)
  }
}

func LoadBmsInDirectory(path string) (bmsobject.BmsDirectory, error) {
  bmsdirectory := bmsobject.NewBmsDirectory()
  bmsdirectory.Path = path
  files, _ := ioutil.ReadDir(path)
  nodifficulty := true
  for _, f := range files {
    if IsBmsPath(f.Name()) {
      var bmsfile bmsobject.BmsFile
      var err error
      noobj := false
      bmspath := filepath.Join(path, f.Name())
      if filepath.Ext(bmspath) == ".bmson" {
        bmsfile, err = LoadBmson(bmspath)
      } else {
        bmsfile, noobj, err = LoadBms(bmspath)
      }
      if err != nil {
        return bmsobject.NewBmsDirectory(), err
      }
      if !noobj {
        if bmsfile.Difficulty == "" {
          bmsfile.Difficulty = getDifficultyFromTitle(bmsfile)
          if bmsfile.Difficulty == "" {
            bmsfile.Difficulty = getDifficultyFromPath(bmsfile)
          }
        }
        if bmsfile.Difficulty != "" {
          nodifficulty = false
        }
        bmsdirectory.Bmsfiles = append(bmsdirectory.Bmsfiles, bmsfile)
      }
    }
  }
  if len(bmsdirectory.Bmsfiles) > 0 {
    bmsdirectory.Name = bmsdirectory.Bmsfiles[0].Title

    // フォルダ内にDifficultyが設定されているBMSが一つもない場合
    if nodifficulty {
      findDifficultyFromDirectory(&bmsdirectory)
    }
  }

  return bmsdirectory, nil
}

func FindBmsInDirectory(path string, bmsdirs *[]bmsobject.BmsDirectory) (error) {
  files, _ := ioutil.ReadDir(path)
  bmsExist := false
  for _, f := range files {
    if IsBmsPath(f.Name()) {
      bmsdirectory, err := LoadBmsInDirectory(path)
      if err != nil {
        return err
      }
      *bmsdirs = append(*bmsdirs, bmsdirectory)
      bmsExist = true
      break
    }
  }
  if !bmsExist {
    for _, f := range files {
      if f.IsDir() {
        err := FindBmsInDirectory(filepath.Join(path, f.Name()), bmsdirs)
        if err != nil {
          return err
        }
      }
    }
  }
  return nil
}
