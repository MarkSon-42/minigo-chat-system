#!/bin/bash

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 프로젝트 루트로 이동
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo -e "${BLUE}=== Chat System Test Suite ===${NC}\n"

# 함수: 테스트 결과 출력
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ $2 PASSED${NC}\n"
    else
        echo -e "${RED}❌ $2 FAILED${NC}\n"
        exit 1
    fi
}

# 함수: 개별 패키지 테스트
test_package() {
    local package=$1
    local package_name=$2
    
    echo -e "${YELLOW}[Testing $package_name]${NC}"
    cd "$SCRIPT_DIR/$package"
    
    if [ ! -f go.mod ]; then
        echo -e "${RED}⚠️  No go.mod found in $package, skipping...${NC}\n"
        return
    fi
    
    go test -v "$@"
    local result=$?
    print_result $result "$package_name"
    
    cd "$SCRIPT_DIR"
}

# 인자 파싱
MODE=${1:-"all"}

case $MODE in
    "quick")
        echo -e "${BLUE}🚀 Quick Test Mode${NC}\n"
        test_package "proxy" "Proxy Tests"
        test_package "backend" "Backend Tests"
        ;;
        
    "race")
        echo -e "${BLUE}🏁 Race Detection Mode${NC}\n"
        test_package "proxy" "Proxy Race Tests" -race
        test_package "backend" "Backend Race Tests" -race
        ;;
        
    "bench")
        echo -e "${BLUE}📊 Benchmark Mode${NC}\n"
        test_package "proxy" "Proxy Benchmarks" -bench=. -benchmem
        test_package "backend" "Backend Benchmarks" -bench=. -benchmem
        ;;
        
    "coverage")
        echo -e "${BLUE}📈 Coverage Mode${NC}\n"
        
        # Proxy coverage
        echo -e "${YELLOW}[Proxy Coverage]${NC}"
        cd "$SCRIPT_DIR/proxy"
        go test -coverprofile=coverage.out -covermode=atomic
        local proxy_result=$?
        if [ $proxy_result -eq 0 ]; then
            go tool cover -func=coverage.out | tail -n 1
            go tool cover -html=coverage.out -o coverage.html
            echo -e "${GREEN}📄 Coverage report: proxy/coverage.html${NC}"
        fi
        print_result $proxy_result "Proxy Coverage"
        
        # Backend coverage
        echo -e "${YELLOW}[Backend Coverage]${NC}"
        cd "$SCRIPT_DIR/backend"
        go test -coverprofile=coverage.out -covermode=atomic
        local backend_result=$?
        if [ $backend_result -eq 0 ]; then
            go tool cover -func=coverage.out | tail -n 1
            go tool cover -html=coverage.out -o coverage.html
            echo -e "${GREEN}📄 Coverage report: backend/coverage.html${NC}"
        fi
        print_result $backend_result "Backend Coverage"
        ;;
        
    "storage")
        echo -e "${BLUE}💾 Storage Tests Only${NC}\n"
        cd "$SCRIPT_DIR/proxy"
        go test -v -run TestStorage
        print_result $? "Storage Tests"
        ;;
        
    "hub")
        echo -e "${BLUE}🔌 Hub Tests Only${NC}\n"
        cd "$SCRIPT_DIR/backend"
        go test -v -run TestHub
        print_result $? "Hub Tests"
        ;;
        
    "deadlock")
        echo -e "${BLUE}🔒 Deadlock Detection Test${NC}\n"
        cd "$SCRIPT_DIR/proxy"
        echo "Testing storage Sync() deadlock fix..."
        go test -v -run TestStorageSync_NoDeadlock -timeout 5s
        print_result $? "Deadlock Test"
        ;;
        
    "all")
        echo -e "${BLUE}🎯 Full Test Suite${NC}\n"
        
        # 1. Quick tests
        echo -e "${YELLOW}Step 1: Running quick tests...${NC}"
        test_package "proxy" "Proxy Tests"
        test_package "backend" "Backend Tests"
        
        # 2. Race detection
        echo -e "${YELLOW}Step 2: Running race detection...${NC}"
        test_package "proxy" "Proxy Race Tests" -race
        test_package "backend" "Backend Race Tests" -race
        
        # 3. Coverage
        echo -e "${YELLOW}Step 3: Generating coverage reports...${NC}"
        cd "$SCRIPT_DIR/proxy"
        go test -coverprofile=coverage.out -covermode=atomic > /dev/null 2>&1
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}Proxy coverage:${NC}"
            go tool cover -func=coverage.out | tail -n 1
        fi
        
        cd "$SCRIPT_DIR/backend"
        go test -coverprofile=coverage.out -covermode=atomic > /dev/null 2>&1
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}Backend coverage:${NC}"
            go tool cover -func=coverage.out | tail -n 1
        fi
        
        echo -e "\n${GREEN}✅ All tests completed successfully!${NC}"
        ;;
        
    "watch")
        echo -e "${BLUE}👀 Watch Mode (requires 'entr')${NC}\n"
        if ! command -v entr &> /dev/null; then
            echo -e "${RED}Error: 'entr' is not installed${NC}"
            echo "Install with: sudo apt install entr"
            exit 1
        fi
        
        echo "Watching for changes in *.go files..."
        find . -name "*.go" | entr -c bash run_tests.sh quick
        ;;
        
    "help"|"-h"|"--help")
        echo "Usage: ./run_tests.sh [MODE]"
        echo ""
        echo "Modes:"
        echo "  quick      - Run basic tests (default)"
        echo "  race       - Run tests with race detector"
        echo "  bench      - Run benchmarks"
        echo "  coverage   - Generate coverage reports"
        echo "  storage    - Run storage tests only"
        echo "  hub        - Run hub tests only"
        echo "  deadlock   - Run deadlock detection test"
        echo "  all        - Run full test suite"
        echo "  watch      - Watch mode (requires entr)"
        echo "  help       - Show this help"
        echo ""
        echo "Examples:"
        echo "  ./run_tests.sh              # Quick test"
        echo "  ./run_tests.sh race         # Race detection"
        echo "  ./run_tests.sh coverage     # Coverage report"
        ;;
        
    *)
        echo -e "${RED}Unknown mode: $MODE${NC}"
        echo "Run './run_tests.sh help' for usage"
        exit 1
        ;;
esac