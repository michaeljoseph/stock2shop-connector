package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	router := httprouter.New()
	router.POST("/products", PutProducts)
	router.GET("/products", GetProducts)
	router.GET("/products/page", GetProductsPage)
	router.DELETE("/products", DeleteProducts)
	router.DELETE("/clean", CleanupDataDir)

	// os.Args[0] is the program
	path := os.Args[1]
	if path == "" {
		log.Fatal("data storage path must be specified as the second argument")
	}

	log.Printf("server stating on port 8080")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":8080"), router))
}

func Response(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Printf("%v", err)
	}
}

type Product struct {
	Name    string    `json:"name"`
	ID      string    `json:"id"`
	Options []Options `json:"options"`
	Images  []Image   `json:"images"`
}

type Options struct {
	Sku string `json:"sku"`
	ID  string `json:"id"`
}

type Image struct {
	Url string `json:"url"`
	ID  string `json:"id"`
}

type Products []Product

type ProductIDs []string

func (p *Product) Validate() error {
	if p.Name == "" {
		err := errors.New("product Name is required")
		return err
	}
	if len(p.Options) == 0 {
		err := errors.New("product must have at least one option")
		return err
	}
	return nil
}

func (p *Products) Validate() error {
	for _, product := range *p {
		err := product.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

func PutProducts(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	dataPath := fmt.Sprintf("%s", os.Args[1])
	products := Products{}

	// populate products with data from request
	err := json.NewDecoder(r.Body).Decode(&products)
	if err != nil {
		Response(w, http.StatusBadRequest, err.Error())
		return
	}

	// validate products
	err = products.Validate()
	if err != nil {
		Response(w, http.StatusBadRequest, err.Error())
		return
	}

	// write products to file, generate id if it is not set
	for i := 0; i < len(products); i++ {
		// ID
		if products[i].ID == "" {
			products[i].ID = strconv.Itoa(int(time.Now().UnixNano()))
		}
		// Variants
		for j := 0; j < len(products[i].Options); j++ {
			if products[i].Options[j].ID == "" {
				products[i].Options[j].ID = strconv.Itoa(int(time.Now().UnixNano()))
			}
		}
		// Images
		for j := 0; j < len(products[i].Images); j++ {
			if products[i].Images[j].ID == "" {
				products[i].Images[j].ID = strconv.Itoa(int(time.Now().UnixNano()))
			}
		}

		data, err := json.MarshalIndent(products[i], "", "    ")
		if err != nil {
			Response(w, http.StatusBadRequest, err.Error())
			return
		}

		err = os.WriteFile(fmt.Sprintf("%s/%s.json", dataPath, products[i].ID), data, 0644)
		if err != nil {
			Response(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	Response(w, http.StatusAccepted, products)
}

func GetProducts(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	dataPath := fmt.Sprintf("%s", os.Args[1])
	getProducts := ProductIDs{}

	// populate products with data from request
	err := json.NewDecoder(r.Body).Decode(&getProducts)
	if err != nil {
		Response(w, http.StatusBadRequest, err.Error())
		return
	}

	// check that there are id's to find
	if len(getProducts) < 1 {
		Response(w, http.StatusAccepted, getProducts)
		return
	}

	// read file data and append to buffer
	buf := new(bytes.Buffer)
	buf.Write([]byte("["))
	for i, id := range getProducts {
		filePath := fmt.Sprintf("%s/%s.json", dataPath, id)
		b, err := os.ReadFile(filePath)
		if err != nil {
			Response(w, http.StatusBadRequest, fmt.Sprintf("unable to read file: %v.json", id))
			return
		}
		buf.Write(b)

		if i < len(getProducts)-1 {
			buf.Write([]byte(","))
		}
	}
	buf.Write([]byte("]"))

	// unmarshal into response type
	products := Products{}
	err = json.Unmarshal(buf.Bytes(), &products)
	if err != nil {
		Response(w, http.StatusAccepted, err.Error())
		return
	}

	Response(w, http.StatusAccepted, products)

}

func GetProductsPage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	dataPath := fmt.Sprintf("%s", os.Args[1])

	// get channel_product_code, in this case it is an offset, default to 0 if not included in url params
	cpc := r.URL.Query().Get("channel_product_code")
	if cpc == "" {
		cpc = "0"
	}

	// get offset, default to 10 if not included in url params
	l := r.URL.Query().Get("limit")
	if l == "" {
		l = "10"
	}
	limit, err := strconv.Atoi(l)
	if err != nil {
		Response(w, http.StatusBadRequest, "invalid limit")
		return
	}

	var files []string
	err = filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".json") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		Response(w, http.StatusBadRequest, err.Error())
		return
	}

	sort.Strings(files)

	if limit == 0 {
		Response(w, http.StatusAccepted, Products{})
		return
	}

	// read file data and append to buffer
	buf := new(bytes.Buffer)
	buf.Write([]byte("["))

	offset := 0
	if cpc != "0" {
		for i := 0; i < len(files); i++ {
			if strings.Contains(files[i], cpc) {
				offset = i + 1
				break
			}
		}
	}

	for i := 0; i < limit; i++ {
		if offset+i >= len(files) {
			break
		}

		b, err := os.ReadFile(files[offset+i])
		if err != nil {
			Response(w, http.StatusBadRequest, fmt.Sprintf("unable to read file: %v", files[offset+i]))
			return
		}
		buf.Write(b)

		if (i+1 < limit) && (offset+i+1 < len(files)) {
			buf.Write([]byte(","))
		}
	}
	buf.Write([]byte("]"))

	// unmarshal into response type
	products := Products{}
	err = json.Unmarshal(buf.Bytes(), &products)
	if err != nil {
		Response(w, http.StatusAccepted, err.Error())
		return
	}

	Response(w, http.StatusAccepted, products)
}

func DeleteProducts(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	dataPath := fmt.Sprintf("%s", os.Args[1])

	ids := ProductIDs{}

	// populate products with data from request
	err := json.NewDecoder(r.Body).Decode(&ids)
	if err != nil {
		Response(w, http.StatusBadRequest, err.Error())
		return
	}

	// check that there are id's to find
	if len(ids) < 1 {
		Response(w, http.StatusAccepted, nil)
		return
	}

	// get all files
	var files []string
	err = filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".json") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		Response(w, http.StatusBadRequest, err.Error())
		return
	}

	// remove files
	for _, id := range ids {
		for _, file := range files {
			if strings.Contains(file, id) {
				err := os.Remove(file)
				if err != nil {
					Response(w, http.StatusBadRequest, err.Error())
					return
				}
			}
		}
	}

	Response(w, http.StatusAccepted, nil)
}

func CleanupDataDir(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	dataPath := fmt.Sprintf("%s", os.Args[1])

	// get all files
	var files []string
	err := filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".json") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		Response(w, http.StatusBadRequest, err.Error())
		return
	}

	// remove files
	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			Response(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	Response(w, http.StatusAccepted, nil)
}
