# Setup Rune GitHub Action

This GitHub Action installs the rune CLI binary into GitHub Actions runner environments.

## Development

### Prerequisites

- Node.js 20 or later
- npm

### Setup

```bash
npm install
```

### Building

```bash
npm run build
```

This will:
1. Compile TypeScript to JavaScript in `lib/`
2. Bundle the application using `@vercel/ncc` to `dist/index.js`

### Testing

```bash
# Run tests
npm test

# Run tests with coverage
npm run test:coverage
```

### Linting

```bash
npm run lint
```

## Project Structure

```
.
├── action.yml              # Action metadata
├── src/
│   ├── main.ts            # Entry point
│   └── install.ts         # Installation logic
├── __tests__/
│   └── install.test.ts    # Tests
├── dist/
│   └── index.js           # Bundled output (committed)
├── package.json
├── tsconfig.json
└── jest.config.js
```

## Usage

See the main repository README for usage instructions.
