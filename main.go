package main

import (
	"fmt"
	"net/http"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	fmt.Fprintf(w, "r degiskeninin bellekteki adresi: %p\n\n", r)
	fmt.Fprintf(w, "adres icindeki veri:\n%+v", r)
}

func main() {
	http.HandleFunc("/api/status", statusHandler)

	fmt.Println("Sunucu 8080 portunda calismaya basladi. http://localhost:8080/api/status adresine gidebilirsin.")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Sunucu baslatilamadi:", err)
	}
}
