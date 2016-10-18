package main

import (
	"github.com/gorilla/mux"

	"net/http"
	"time"
	"log"

	opentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"fmt"
	"os"
	"github.com/opentracing/opentracing-go/ext"
)

const zipkinHTTPEndpoint = "http://172.16.6.213:9411/api/v1/spans"
const debugMode = true

const set_of_products = `
[
    {
        "id": 2,
        "name": "An ice sculpture",
        "price": 12.50,
        "tags": ["cold", "ice"],
        "dimensions": {
            "length": 7.0,
            "width": 12.0,
            "height": 9.5
        },
        "warehouseLocation": {
            "latitude": -78.75,
            "longitude": 20.4
        }
    },
    {
        "id": 3,
        "name": "A blue mouse",
        "price": 25.50,
            "dimensions": {
            "length": 3.1,
            "width": 1.0,
            "height": 1.0
        },
        "warehouseLocation": {
            "latitude": 54.4,
            "longitude": -32.7
        }
    }
]
`


// Tracker Handler ( middleware )
// Decorate the http handler to provide additional resources for the span tracking.
func TrackerHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to join to a trace propagated in `req`.

		wireContext, err := opentracing.GlobalTracer().Extract(
			opentracing.TextMap,
			opentracing.HTTPHeadersCarrier(r.Header),
		)
		if err != nil {
			log.Println("error encountered while trying to extract span: %s", err)
		}

		span := opentracing.StartSpan(r.RequestURI, ext.RPCServerOption(wireContext))
		defer span.Finish()
		// store span in context
		ctx := opentracing.ContextWithSpan(r.Context(), span)
		// update request context to include our new span
		r = r.WithContext(ctx)

		h.ServeHTTP(w,r)
	})
}



func ProductHandler(res http.ResponseWriter, r  *http.Request){
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(set_of_products))
}


func main() {

	collector, err := zipkin.NewHTTPCollector(zipkinHTTPEndpoint)
	if err != nil {
		fmt.Printf("unable to create Zipkin HTTP collector: %+v", err)
		os.Exit(-1)
	}
	recorder := zipkin.NewRecorder(collector, debugMode, "localhost:8000", "productService")

	// Create our tracer
	// Set to true for RPC style spans
	tracer, err := zipkin.NewTracer(
		recorder, zipkin.ClientServerSameSpan(true),
	)

	if err != nil {
		fmt.Printf("unable to create Zipkin tracer: %+v", err)
		os.Exit(-1)
	}

	opentracing.InitGlobalTracer(tracer)

	r := mux.NewRouter()
	r.Methods(http.MethodGet).Path("/products").HandlerFunc(ProductHandler)

	srv := &http.Server{
		Handler:      TrackerHandler(r),
		Addr:         "127.0.0.1:8000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}


	log.Fatal(srv.ListenAndServe())
}

