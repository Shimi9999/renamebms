package bmsobject

type BmsFile struct {
  Path string
  Title string
  Subtitle string
  Playlevel string
  Difficulty string
  Artist string
  Genre string
  Keymode int // 5, 7, 9, 10, 14, 24, 48
  Md5 string
  Sha256 string
}

func NewBmsFile() BmsFile {
  var bf BmsFile
  bf.Keymode = 7
  return bf
}

type BmsDirectory struct {
  Path string
  Name string
  Bmsfiles []BmsFile
}

func NewBmsDirectory() BmsDirectory {
  var bd BmsDirectory
  bd.Bmsfiles = make([]BmsFile, 0)
  return bd
}
