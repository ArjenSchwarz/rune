import * as fs from 'fs';
import * as path from 'path';

describe('action.yml', () => {
  const metadata = fs.readFileSync(path.join(__dirname, '..', 'action.yml'), 'utf8');

  it('runs on the node24 runtime', () => {
    const using = metadata.match(/^\s+using:\s*'?([^'\s]+)'?\s*$/m);
    expect(using?.[1]).toBe('node24');
  });

  it('runs the bundled entry point', () => {
    expect(metadata).toMatch(/^\s+main:\s*'dist\/index\.js'\s*$/m);
  });
});
