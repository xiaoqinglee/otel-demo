package main

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"io"
	"log"
	"os"
	"os/signal"
)

func main() {
	l := log.New(os.Stdout, "", 0)

	//// Write telemetry data to a file.
	//f, err := os.Create("traces.txt")
	//if err != nil {
	//	l.Fatal(err)
	//}
	//defer f.Close()
	//
	//exp, err := newWriterExporter(f)
	//if err != nil {
	//	l.Fatal(err)
	//}

	JaegerURI := "http://localhost:14268/api/traces"
	exp, err := newJaegerExporter(JaegerURI)
	if err != nil {
		l.Fatal(err)
	}

	//Installing a Tracer Provider
	//
	//You have your application instrumented to produce telemetry data
	//and you have an exporter to send that data to the console,
	//but how are they connected?
	//This is where the TracerProvider is used.
	//这是一个集中点，instrumentation 将从这里获取 Tracer 然后将来自这些 Tracer 的指标数据汇聚到导出管道。
	//It is a centralized point
	//where instrumentation will get a Tracer from and funnels the telemetry data from these Tracers to export pipelines.
	//
	//The pipelines that receive and ultimately transmit data to exporters are called SpanProcessors.
	//A TracerProvider can be configured to have multiple span processors,
	//but for this example you will only need to configure only one.
	//
	// Tracer 负责在程序现场产生指标数据, Exporter 负责将数据持久化到相应的位置.
	// TracerProvider 连接了 Tracer 和 Exporter.
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(newResource()),
	)
	//Finally, with the TracerProvider created, you are deferring a function to flush and stop it.
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			l.Fatal(err)
		}
	}()
	//Registering it as the global OpenTelemetry TracerProvider.
	//This last step, registering the TracerProvider globally,
	//is what will connect that instrumentation’s Tracer with this TracerProvider.
	otel.SetTracerProvider(tp)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	errCh := make(chan error)
	app := NewApp(os.Stdin, l)
	go func() {
		errCh <- app.Run(context.Background())
	}()

	select {
	case <-sigCh:
		l.Println("\ngoodbye")
		return
	case err := <-errCh:
		if err != nil {
			l.Fatal(err)
		}
	}
}

// newWriterExporter returns a writer exporter.
func newWriterExporter(w io.Writer) (tracesdk.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		// Use human-readable output.
		stdouttrace.WithPrettyPrint(),
		// Do not print timestamps for the demo.
		stdouttrace.WithoutTimestamps(),
	)
}

// newJaegerExporter returns a jaeger exporter.
func newJaegerExporter(url string) (tracesdk.SpanExporter, error) {
	// Create the Jaeger exporter
	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
}

//Creating a Resource
//
//Telemetry data can be crucial to solving issues with a service.
//The catch is, you need a way to identify what service, or even what service instance, that data is coming from.
//OpenTelemetry 使用 Resource 来代表产生指标数据的实体.
//OpenTelemetry uses a Resource to represent the entity producing telemetry.

// newResource returns a resource describing this application.
func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("fib"),
			semconv.ServiceVersion("v0.1.0"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}
