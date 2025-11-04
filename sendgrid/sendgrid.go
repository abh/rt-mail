package sendgrid

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.askask.com/rt-mail/rt"
)

type Sendgrid struct {
	RT *rt.RT
}

func (sg *Sendgrid) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/sendgrid/mx", sg.ReceiveHandler)
}

type Envelope struct {
	From string
	To   []string
}

func (sg *Sendgrid) ReceiveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Printf("POST to '%s': %#v\n\n", r.URL.String(), r)

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*50)
	defer r.Body.Close()
	r.ParseMultipartForm(64 << 20)

	form := r.PostForm
	// for k, v := range form {
	// 	fmt.Printf("data %s: %s \n", k, v)
	// }

	to := form.Get("envelope")

	var result Envelope

	json.Unmarshal([]byte(to), &result)

	fmt.Printf("envelope.to %s: \n", result.To)

	body := form.Get("email")

	allNotFound := true
	var err error

	// fmt.Printf("body: %s \n", body)

	for _, email := range result.To {
		fmt.Printf("to: %s \n", email)
		err = sg.RT.Postmail(email, body)
		if err != nil {
			fmt.Printf("post error: %s \n", err)
			if err, ok := err.(*rt.Error); ok {
				if err.NotFound {
					fmt.Printf("inserting email %s failed, address not setup\n", email)
					continue
				}
				w.WriteHeader(http.StatusServiceUnavailable)
				break
			}
			continue
		}
		fmt.Printf("inserting email succeeded %s\n", email)
		allNotFound = false
	}

	if allNotFound == true {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
