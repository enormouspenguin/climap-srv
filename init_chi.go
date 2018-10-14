package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
	"github.com/rwcarlsen/goexif/exif"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	MAX_MULTIPART_PARSE_MEM = 10 * 1024 * 1024 //10 MB
)

var rtr chi.Router

const (
	URL_PARAM_IMG_ID_NAME = "imgID"
)

var (
	_PIC_URL_PATH string
)

func init_chi() {
	rtr = chi.NewRouter()
	rtr.Use(middleware.RedirectSlashes)

	_PIC_URL_PATH := path.Join("/", _STATIC_SUBDIR_NM, _PIC_SUBDIR_NM)
	// fmt.Println(_PIC_URL_PATH)
	// fmt.Println(_PIC_PATH)
	internalFS := newDirFS(_PIC_PATH)
	// fmt.Println(internalFS.Open("/9a6e8f9c76c2adaa0dde68968e5eeee56f0db912"))
	rtr.Get(_PIC_URL_PATH+"*", http.StripPrefix(_PIC_URL_PATH, http.FileServer(internalFS)).ServeHTTP)

	rtr.Route("/api/", func(rtr chi.Router) {
		rtr.Get("/search", Search)

		rtr.Route("/img", func(rtr chi.Router) {
			rtr.Post("/", UploadImg)
			rtr.Get("/{"+URL_PARAM_IMG_ID_NAME+"}", GetImg)
		})

		rtr.Get("/tag", ListAllTags)
	})

}

func Search(w http.ResponseWriter, r *http.Request) {
	strSWlon := r.URL.Query().Get("swlon")
	SWX, err := strconv.ParseFloat(strSWlon, 64)
	if err != nil {
		http.Error(w, "unexpected SW longitude value: "+strSWlon, http.StatusBadRequest)
		return
	}

	strSWlat := r.URL.Query().Get("swlat")
	SWY, err := strconv.ParseFloat(strSWlat, 64)
	if err != nil {
		http.Error(w, "unexpected SW latitude value: "+strSWlat, http.StatusBadRequest)
		return
	}

	strNElon := r.URL.Query().Get("nelon")
	NEX, err := strconv.ParseFloat(strNElon, 64)
	if err != nil {
		http.Error(w, "unexpected NE longitude value: "+strNElon, http.StatusBadRequest)
		return
	}

	strNElat := r.URL.Query().Get("nelat")
	NEY, err := strconv.ParseFloat(strNElat, 64)
	if err != nil {
		http.Error(w, "unexpected NE latitude value: "+strNElat, http.StatusBadRequest)
		return
	}

	queryStr := `SELECT loc, tag, dsc, url, hash, added_at
	FROM ` + TB_IMG + `
	WHERE loc <@ box(point($1, $2), point($3, $4))`

	var rows *pgx.Rows
	args := []interface{}{NEX, NEY, SWX, SWY}

	if tag := r.URL.Query().Get("tag"); tag != "" {
		queryStr += " && tag=$5"
		args = append(args, tag)
	}

	rows, err = cpool.Query(queryStr+";", args...)

	imgs, err := toObjs(rows, r)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	w.Header().Set("Content-Type", "application/json")

	err = encoder.Encode(imgs)
	if err != nil {
		fmt.Println(err)
	}
}

func UploadImg(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(MAX_MULTIPART_PARSE_MEM)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	Tag := r.FormValue("tag")
	Dsc := r.FormValue("dsc")
	URL := r.FormValue("url")

	var (
		Loc  *Point
		Hash string
	)

	if chkImgURL(URL) {
		//go get the image (without storing locally) and calculate lat, lon and hash
		resp, err := http.Get(URL)
		defer resp.Body.Close()

		Hash, Loc, err = saveImg(resp.Body, ioutil.Discard)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		img, _, err := r.FormFile("img")
		defer img.Close()

		if err != nil {
			http.Error(w, "must supply image", http.StatusBadRequest)
			return
		}

		dst, err := ioutil.TempFile(_PIC_PATH, "pending_")
		if err != nil {
			fmt.Println("Unable to create temp file:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		Hash, Loc, err = saveImg(img, dst)
		if err != nil {
			fmt.Println("Unable to save:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		dstName := dst.Name()
		dst.Close()

		err = os.Rename(dstName, filepath.Join(_PIC_PATH, Hash))
		if err != nil {
			fmt.Println("Unable to rename:", err)
			fmt.Println("Removing the temp file:", os.Remove(dst.Name()))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		URL = path.Join(_PIC_URL_PATH, Hash)
	}

	if Loc == nil { //image has no lat/lon information
		Loc = &Point{}

		strlon := r.FormValue("lon")
		Loc.Lon, err = strconv.ParseFloat(strlon, 64)
		if err != nil {
			http.Error(w, "unable to extract longitude value from image and unexpected longitude fallback form value: "+strlon, http.StatusBadRequest)
			return
		}

		strlat := r.FormValue("lat")
		Loc.Lat, err = strconv.ParseFloat(strlat, 64)
		if err != nil {
			http.Error(w, "unable to extract longitude value from image and unexpected latitude fallback form value: "+strlat, http.StatusBadRequest)
			return
		}
	}

	var id int64

	err = cpool.QueryRow(`
INSERT INTO `+TB_IMG+`(loc, tag, dsc, url, hash)
VALUES (point($1, $2), $3, $4, $5, $6)
RETURNING id;`,
		Loc.Lon, Loc.Lat, Tag, Dsc, URL, Hash).Scan(&id)
	if err != nil {
		os.Remove(filepath.Join(_PIC_PATH, Hash))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println(cpool.Exec(`
		INSERT INTO `+TB_TAG+` (tag)
		VALUES ($1);`, Tag))

	w.Write([]byte(strconv.Itoa(int(id))))
}

func chkImgURL(URL string) bool {
	_, err := url.ParseRequestURI(URL)
	return err == nil
}

func saveImg(img io.Reader, dst io.Writer) (hash string, p *Point, err error) {
	hashWrt := sha1.New()

	img = io.TeeReader(img, io.MultiWriter(dst, hashWrt))

	exifInfo, err := exif.Decode(img)
	if err != nil {
		return
	}

	_, err = io.Copy(ioutil.Discard, img)
	if err != nil {
		return
	}

	hash = hex.EncodeToString(hashWrt.Sum(nil))

	lat, lon, err := exifInfo.LatLong()
	if err == nil {
		p = &Point{
			Lat: lat,
			Lon: lon,
		}
	} else { //err != nil
		//ignore EXIF error
		err = nil
	}

	return
}

type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Img struct {
	Id      int64     `json:"id"`
	Loc     Point     `json:"loc"`
	Tag     string    `json:"tag"`
	Dsc     string    `json:"dsc"`
	URL     string    `json:"url"`
	Hash    string    `json:"hash"`
	AddedAt time.Time `json:"addedat"`
}

func GetImg(w http.ResponseWriter, r *http.Request) {
	var (
		img    Img
		PgxPnt pgtype.Point
	)

	id, err := strconv.Atoi(chi.URLParam(r, URL_PARAM_IMG_ID_NAME))
	img.Id = int64(id)

	err = cpool.QueryRow(`
	SELECT loc, tag, dsc, url, hash, added_at FROM `+TB_IMG+` WHERE id=$1;
	`, int64(id)).Scan(&PgxPnt, &img.Tag, &img.Dsc, &img.URL, &img.Hash, &img.AddedAt)
	if err != nil {
		http.Error(w, "unable to find the image", http.StatusNotFound)
		return
	}

	img.Loc.Lon = PgxPnt.P.X
	img.Loc.Lat = PgxPnt.P.Y

	if !strings.HasPrefix(img.URL, "http") {
		img.URL = r.URL.Scheme + "://" + r.Host + img.URL
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	w.Header().Set("Content-Type", "application/json")

	err = encoder.Encode(img)
	if err != nil {
		fmt.Println(err)
	}
}

//SELECT loc, tag, dsc, url, hash, added_at
func toObjs(rows *pgx.Rows, r *http.Request) (imgs []Img, err error) {
	defer rows.Close()

	imgs = []Img{} //encode to JSON "[]" instead of "null"

	var (
		img    Img
		PgxPnt pgtype.Point
	)

	for rows.Next() {
		err = rows.Scan(&PgxPnt, &img.Tag, &img.Dsc, &img.URL, &img.Hash, &img.AddedAt)
		if err != nil {
			return
		}

		img.Loc.Lon = PgxPnt.P.X
		img.Loc.Lat = PgxPnt.P.Y

		if !strings.HasPrefix(img.URL, "http") {
			img.URL = r.URL.Scheme + "://" + r.Host + img.URL
		}

		imgs = append(imgs, img)
	}

	return
}

func ListAllTags(w http.ResponseWriter, r *http.Request) {
	rows, err := cpool.Query(`SELECT tag FROM ` + TB_TAG + `;`)
	defer rows.Close()

	tag := ""

	tags := []string{} //encode to JSON "[]" instead of "null"

	for rows.Next() {
		err = rows.Scan(tag)
		tags = append(tags, tag)
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	w.Header().Set("Content-Type", "application/json")

	err = encoder.Encode(tags)
	if err != nil {
		fmt.Println(err)
	}
}
