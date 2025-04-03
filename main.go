package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"

	"github.com/optiop/grafana-whatsapp-webhook/pkg/service"
	"github.com/optiop/grafana-whatsapp-webhook/pkg/whatsapp"
)

func main() {
	var wg sync.WaitGroup

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	wg.Add(2)
	ws := whatsapp.New(ctx, &wg)
	service.Run(ctx, ws, &wg)

	wg.Wait()
	fmt.Println("Program terminated gracefully")
}
