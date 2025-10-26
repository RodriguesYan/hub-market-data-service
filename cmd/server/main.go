package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RodriguesYan/hub-market-data-service/internal/application/usecase"
	"github.com/RodriguesYan/hub-market-data-service/internal/config"
	"github.com/RodriguesYan/hub-market-data-service/internal/infrastructure/cache"
	"github.com/RodriguesYan/hub-market-data-service/internal/infrastructure/persistence"
	grpcServer "github.com/RodriguesYan/hub-market-data-service/internal/presentation/grpc"
	cacheHandler "github.com/RodriguesYan/hub-market-data-service/pkg/cache"
	"github.com/RodriguesYan/hub-market-data-service/pkg/database"
	pb "github.com/RodriguesYan/hub-proto-contracts/monolith"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log.Println("Starting Market Data Service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := initializeDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	redisClient := initializeRedis(cfg)
	defer redisClient.Close()

	marketDataRepo := persistence.NewMarketDataRepository(db)

	cacheClient := cacheHandler.NewRedisCacheHandler(redisClient)
	cachedMarketDataRepo := cache.NewMarketDataCacheRepository(
		marketDataRepo,
		cacheClient,
		5*time.Minute,
	)

	getMarketDataUsecase := usecase.NewGetMarketDataUseCase(cachedMarketDataRepo)

	grpcSrv := startGRPCServer(cfg, getMarketDataUsecase)

	log.Printf("Market Data Service started successfully")
	log.Printf("gRPC server listening on port %s", cfg.GRPC.Port)

	waitForShutdown(grpcSrv)
}

func initializeDatabase(cfg *config.Config) (database.Database, error) {
	log.Printf("Connecting to database at %s:%s...", cfg.Database.Host, cfg.Database.Port)

	sqlxDB, err := sqlx.Connect("postgres", cfg.GetDatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlxDB.SetMaxOpenConns(25)
	sqlxDB.SetMaxIdleConns(5)
	sqlxDB.SetConnMaxLifetime(5 * time.Minute)

	if err := sqlxDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established successfully")

	return database.NewSQLXDatabase(sqlxDB), nil
}

func initializeRedis(cfg *config.Config) *redis.Client {
	log.Printf("Connecting to Redis at %s...", cfg.GetRedisAddr())

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Redis connection established successfully")

	return client
}

func startGRPCServer(cfg *config.Config, getMarketDataUsecase usecase.IGetMarketDataUsecase) *grpc.Server {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPC.Port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPC.Port, err)
	}

	grpcSrv := grpc.NewServer()

	marketDataServer := grpcServer.NewMarketDataGRPCServer(getMarketDataUsecase)
	pb.RegisterMarketDataServiceServer(grpcSrv, marketDataServer)

	reflection.Register(grpcSrv)

	go func() {
		log.Printf("gRPC server starting on port %s", cfg.GRPC.Port)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	return grpcSrv
}

func waitForShutdown(grpcSrv *grpc.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Printf("Received signal %v, initiating graceful shutdown...", sig)

	log.Println("Stopping gRPC server...")
	grpcSrv.GracefulStop()

	log.Println("Market Data Service shut down successfully")
}
