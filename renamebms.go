package main

import (
  "fmt"
  "os"
  "flag"
  "strings"
  "regexp"
  "io/ioutil"
  "path/filepath"
  "unicode/utf8"

  "./bmsobject"
  "./bmsloader"
)

func main() {
  var (
    format = flag.String("format", "[%artist%] %title%", "Rename format")
    pre = flag.Bool("pre", false, "If true, only print and don't rename")
  )
  flag.Parse()

  if len(flag.Args()) >= 2 {
    fmt.Println("Usage: renamebms [-format <format>] [-pre=<bool>] [rootdirpath]")
    os.Exit(1)
  }

  var path string
  if len(flag.Args()) == 0 {
    path = "./"
  } else {
    path = flag.Arg(0)
  }
  fInfo, err := os.Stat(path)
  if err != nil {
    fmt.Println("Error: Path is wrong:", err.Error())
    os.Exit(1)
  }
  if !fInfo.IsDir() {
    fmt.Println("Error: The enterd path is not directory.")
    os.Exit(1)
  }

  pwd, _ := os.Getwd()
  abspath, err := filepath.Abs(path)
  if err != nil {
    fmt.Println("Error: Getting absolute path is wrong:", err.Error())
    os.Exit(1)
  }
  if filepath.Clean(abspath) == filepath.Clean(pwd) {
    files, _ := ioutil.ReadDir(path)
    for _, f := range files {
      if bmsloader.IsBmsPath(f.Name()) {
        fmt.Println("Error: Enterd directory is current directory and it has BMS files.")
        os.Exit(1)
      }
    }
  } else if strings.Contains(filepath.Clean(pwd), filepath.Clean(abspath)) {
    fmt.Println("Error: Enterd directory is parent directory.")
    os.Exit(1)
  }

  bmsdirs := make([]bmsobject.BmsDirectory, 0)
  err = bmsloader.FindBmsInDirectory(path, &bmsdirs)
  if err != nil {
    fmt.Println(err.Error())
    os.Exit(1)
  }
  if len(bmsdirs) == 0 {
    fmt.Println("Error: No BMS files")
    os.Exit(1)
  }

  err = renameBmsDir(path, &bmsdirs, *format, *pre)
  if err != nil {
    fmt.Println(err.Error())
    os.Exit(1)
  }
}

func renameBmsDir(rootpath string, bmsdirs *[]bmsobject.BmsDirectory, format string, noRename bool) error {
  errCount := 0
  for _, bmsdir := range *bmsdirs {
    title, artist := getProperTitleAndArtist(&bmsdir)
    if title == "" && artist == "" {
      fmt.Println("Title and artist are empty:", bmsdir.Path)
      errCount++
      continue
    }

    artist = strings.Replace(artist, " / ", "/", -1)
    renamePath := replaceBannedChar(
      strings.Replace(strings.Replace(format, "%artist%", artist, -1), "%title%", title, -1))
    if filepath.Dir(bmsdir.Path) != "." {
      renamePath = filepath.Clean(filepath.Dir(bmsdir.Path) + "/" + renamePath)
    }
    fmt.Println("rename:", renamePath, "<=", bmsdir.Path)

    if !noRename {
      err := os.Rename(bmsdir.Path, renamePath)
      if err != nil {
        return fmt.Errorf("rename error: " + err.Error())
      }
    }
  }
  if errCount > 0 {
    fmt.Printf("Error %d\n", errCount)
  }
  if !noRename {
    fmt.Println("All renamed")
  }

  return nil
}

func replaceBannedChar(text string) string {
  bans := [][]string{{"\\", "￥"}, {"/", "／"}, {":", "："}, {"*", "＊"}, {"?", "？"},
    {"\"", "”"}, {"<", "＜"}, {">", "＞"}, {"|", "｜"}}
  for _, ban := range bans {
    text = strings.Replace(text, ban[0], ban[1], -1)
  }
  return text
}

func getProperTitleAndArtist(bmsdir *bmsobject.BmsDirectory) (string, string) {
  if len(bmsdir.Bmsfiles) == 0 {
    return "", ""
  } else if len(bmsdir.Bmsfiles) == 1 {
    t := trimBrackets(bmsdir.Bmsfiles[0].Title)
    return t, getProperArtist(t, bmsdir)
  }

  titles := []string{}
  shortest := ""
  for i, bmsfile := range bmsdir.Bmsfiles {
    title := trimBothSpace(bmsfile.Title)
    if utf8.RuneCountInString(title) == 0 {
      continue
    }
    titles = append(titles, title)
    if i == 0 {
      shortest = title
    } else if utf8.RuneCountInString(title) < utf8.RuneCountInString(shortest) {
      shortest = title
    }
  }
  if shortest == "" {
    return "", getProperArtist("", bmsdir)
  }

  i := 0
  for ;; i++ {
    headmap := make(map[string]([]string), len(titles))
    for _, title := range titles {
      head := "end"
      if utf8.RuneCountInString(title) > i {
        head = string(([]rune(title))[i:i + 1])
      }
      headmap[head] = append(headmap[head], title)
    }
    maxkey := "init"
    for key, value := range headmap {
      if maxkey == "init" {
        maxkey = key
      } else if len(headmap[maxkey]) < len(value) {
        maxkey = key
      }
    }
    if maxkey == "end" || len(headmap[maxkey]) < 2 ||
    float64(len(headmap[maxkey])) < float64(len(bmsdir.Bmsfiles)) * 0.5 {
      break
    } else {
      titles = headmap[maxkey]
    }
  }

  properTitle := string(([]rune(titles[0]))[:i])

  return trimOneBracket(properTitle, bmsdir), getProperArtist(properTitle, bmsdir)
}

func getProperArtist(properTitle string, bmsdir *bmsobject.BmsDirectory) string {
  var shortest string
  initialized := false
  for _, bmsfile := range bmsdir.Bmsfiles {
    if ((properTitle != "" && strings.HasPrefix(bmsfile.Title, properTitle)) ||
    (properTitle == "" && bmsfile.Title == "")) && bmsfile.Artist != "" {
      if !initialized {
        shortest = bmsfile.Artist
        initialized = true
      } else {
        if utf8.RuneCountInString(bmsfile.Artist) < utf8.RuneCountInString(shortest) {
          shortest = bmsfile.Artist
        }
      }
    }
  }
  return trimBothSpace(shortest)
}

func trimBothSpace(str string) string {
   return strings.Trim(strings.TrimSpace(str), "　")
}

func trimBrackets(title string) string {
  title = trimBothSpace(title)
  regBrackets := [][]string{{`\[`,`\]`}, {`\(`,`\)`}, {"-","-"}, {`【`,`】`}, {"<", ">"}}
  keywords := []string{"beginner", "normal", "hyper", "another", "insane", "easy", "hard", "ex",
    "ez", "nm", "hd", "mx", "sc",
    "5k", "7k", "9k", "10k", "14k"}
  predifs := []string{/*"",*/ "sp", "dp", "5", "7", "9", "10", "14"}
  difs := []string{"b", "n", "h", "a", "i"} // dif単体は完全一致のみで使用する
  for _, predif := range predifs {
    for _, dif := range difs {
      keywords = append(keywords, predif + dif)
    }
  }

  for _, bracket := range regBrackets {
    for _, keyword := range keywords {
      r := regexp.MustCompile(bracket[0] + "[^" + bracket[0] + "]*" + keyword + ".*" + bracket[1] + "$")
      index := r.FindStringIndex(strings.ToLower(title))
      if index != nil {
        return trimBothSpace(title[:index[0]])
      }
    }
    for _, dif := range difs {
      r := regexp.MustCompile(bracket[0] + dif + bracket[1] + "$")
      index := r.FindStringIndex(strings.ToLower(title))
      if index != nil {
        return trimBothSpace(title[:index[0]])
      }
    }
  }
  return title
}

func trimOneBracket(properTitle string, bmsdir *bmsobject.BmsDirectory) string {
  properTitle = trimBothSpace(properTitle)
  brackets := [][]string{{"[","]"}, {"(",")"}, {`【`,`】`}, {"<", ">"}}

  contains := false
  for _, bracket := range brackets {
    openIndex := strings.LastIndex(properTitle, bracket[0])
    closeIndex := strings.LastIndex(properTitle, bracket[1])
    if (openIndex != -1 && openIndex > closeIndex) {
      contains = true
      break
    }
  }
  if strings.Count(properTitle, "-") % 2 == 1 {
    contains = true
  }
  if !contains {
    return properTitle
  }

  regBrackets := [][]string{{`\[`,`\]`}, {`\(`,`\)`}, {"-","-"}, {`【`,`】`}, {"<", ">"}}
  for _, bmsfile := range bmsdir.Bmsfiles {
    if strings.HasPrefix(bmsfile.Title, properTitle) {
      for _, bracket := range regBrackets {
        r := regexp.MustCompile(bracket[0] + "[^" + bracket[0] + "]*" + bracket[1] + "$")
        index := r.FindStringIndex(strings.ToLower(bmsfile.Title))
        if index != nil {
          return trimBothSpace(bmsfile.Title[:index[0]])
        }
      }
    }
  }
  return properTitle
}
