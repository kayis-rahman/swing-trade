# Local Go stack with frontend

Start backend:

```bash
cd /Users/kayisrahman/.treehouse/swing-trade-3a2f1e/1/swing-trade
./bin/go-run
```

Start frontend:

```bash
cd /Users/kayisrahman/.treehouse/swing-trade-3a2f1e/1/swing-trade/dashboard
yarn dev
```

Evidence that dashboard data comes from Go: `http://localhost:3003/api/health` returns `systemInfo.javaVersion="go1.27.3"`, identical to direct `http://127.0.0.1:8081/api/health`.
