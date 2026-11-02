import { resolveVersion, getPlatformAsset, verifyChecksum, installRune } from '../src/install';
import { getOctokit } from '@actions/github';
import * as fs from 'fs';
import { HTTPError } from '@actions/tool-cache';

// Mock dependencies
jest.mock('@actions/github');
jest.mock('@actions/core');
// Keep the real HTTPError so the 404 fallback check exercises actual instanceof/status logic
jest.mock('@actions/tool-cache', () => ({
  ...jest.createMockFromModule<typeof import('@actions/tool-cache')>('@actions/tool-cache'),
  HTTPError: jest.requireActual<typeof import('@actions/tool-cache')>('@actions/tool-cache').HTTPError
}));
jest.mock('@actions/exec');

// Mock fs with proper promises support
jest.mock('fs', () => {
  const actualFs = jest.requireActual<typeof fs>('fs');
  return {
    ...actualFs,
    promises: {
      readFile: jest.fn()
    },
    createReadStream: jest.fn()
  };
});

describe('resolveVersion', () => {
  const mockOctokit = {
    rest: {
      repos: {
        getLatestRelease: jest.fn(),
        getReleaseByTag: jest.fn()
      }
    }
  };

  beforeEach(() => {
    jest.clearAllMocks();
    (getOctokit as jest.Mock).mockReturnValue(mockOctokit);
  });

  test('handles latest version resolution', async () => {
    mockOctokit.rest.repos.getLatestRelease.mockResolvedValue({
      data: { tag_name: 'v1.0.0' }
    });

    const version = await resolveVersion('latest', 'test-token');

    expect(version).toBe('1.0.0');
    expect(mockOctokit.rest.repos.getLatestRelease).toHaveBeenCalledWith({
      owner: 'ArjenSchwarz',
      repo: 'rune'
    });
  });

  test('strips v prefix from tag_name when resolving latest', async () => {
    mockOctokit.rest.repos.getLatestRelease.mockResolvedValue({
      data: { tag_name: 'v2.5.3' }
    });

    const version = await resolveVersion('latest', 'test-token');

    expect(version).toBe('2.5.3');
  });

  test('accepts exact version with v prefix', async () => {
    mockOctokit.rest.repos.getReleaseByTag.mockResolvedValue({
      data: { tag_name: 'v1.0.0' }
    });

    const version = await resolveVersion('v1.0.0', 'test-token');

    expect(version).toBe('1.0.0');
    expect(mockOctokit.rest.repos.getReleaseByTag).toHaveBeenCalledWith({
      owner: 'ArjenSchwarz',
      repo: 'rune',
      tag: 'v1.0.0'
    });
  });

  test('accepts exact version without v prefix', async () => {
    mockOctokit.rest.repos.getReleaseByTag.mockResolvedValue({
      data: { tag_name: 'v1.0.0' }
    });

    const version = await resolveVersion('1.0.0', 'test-token');

    expect(version).toBe('1.0.0');
    expect(mockOctokit.rest.repos.getReleaseByTag).toHaveBeenCalledWith({
      owner: 'ArjenSchwarz',
      repo: 'rune',
      tag: 'v1.0.0'
    });
  });

  test('throws clear error for 404 non-existent version', async () => {
    const error: any = new Error('Not Found');
    error.status = 404;
    mockOctokit.rest.repos.getReleaseByTag.mockRejectedValue(error);

    await expect(resolveVersion('99.99.99', 'test-token'))
      .rejects
      .toThrow('Version 99.99.99 not found');

    await expect(resolveVersion('99.99.99', 'test-token'))
      .rejects
      .toThrow('Check available versions at: https://github.com/ArjenSchwarz/rune/releases');
  });

  test('re-throws non-404 errors', async () => {
    const error = new Error('API rate limit exceeded');
    mockOctokit.rest.repos.getReleaseByTag.mockRejectedValue(error);

    await expect(resolveVersion('1.0.0', 'test-token'))
      .rejects
      .toThrow('API rate limit exceeded');
  });
});

describe('getPlatformAsset', () => {
  const originalPlatform = process.platform;
  const originalArch = process.arch;

  afterEach(() => {
    Object.defineProperty(process, 'platform', { value: originalPlatform, configurable: true });
    Object.defineProperty(process, 'arch', { value: originalArch, configurable: true });
  });

  test('generates correct asset name for Linux amd64', () => {
    Object.defineProperty(process, 'platform', { value: 'linux', configurable: true });
    Object.defineProperty(process, 'arch', { value: 'x64', configurable: true });

    const result = getPlatformAsset('1.0.0');

    expect(result.assetName).toBe('rune-v1.0.0-linux-amd64.tar.gz');
    expect(result.isWindows).toBe(false);
  });

  test('generates correct asset name for Linux arm64', () => {
    Object.defineProperty(process, 'platform', { value: 'linux', configurable: true });
    Object.defineProperty(process, 'arch', { value: 'arm64', configurable: true });

    const result = getPlatformAsset('1.0.0');

    expect(result.assetName).toBe('rune-v1.0.0-linux-arm64.tar.gz');
    expect(result.isWindows).toBe(false);
  });

  test('generates correct asset name for macOS amd64', () => {
    Object.defineProperty(process, 'platform', { value: 'darwin', configurable: true });
    Object.defineProperty(process, 'arch', { value: 'x64', configurable: true });

    const result = getPlatformAsset('2.5.3');

    expect(result.assetName).toBe('rune-v2.5.3-darwin-amd64.tar.gz');
    expect(result.isWindows).toBe(false);
  });

  test('generates correct asset name for macOS arm64', () => {
    Object.defineProperty(process, 'platform', { value: 'darwin', configurable: true });
    Object.defineProperty(process, 'arch', { value: 'arm64', configurable: true });

    const result = getPlatformAsset('1.0.0');

    expect(result.assetName).toBe('rune-v1.0.0-darwin-arm64.tar.gz');
    expect(result.isWindows).toBe(false);
  });

  test('generates correct asset name for Windows amd64', () => {
    Object.defineProperty(process, 'platform', { value: 'win32', configurable: true });
    Object.defineProperty(process, 'arch', { value: 'x64', configurable: true });

    const result = getPlatformAsset('1.0.0');

    expect(result.assetName).toBe('rune-v1.0.0-windows-amd64.zip');
    expect(result.isWindows).toBe(true);
  });

  test('generates correct asset name for Windows arm64', () => {
    Object.defineProperty(process, 'platform', { value: 'win32', configurable: true });
    Object.defineProperty(process, 'arch', { value: 'arm64', configurable: true });

    const result = getPlatformAsset('1.0.0');

    expect(result.assetName).toBe('rune-v1.0.0-windows-arm64.zip');
    expect(result.isWindows).toBe(true);
  });

  test('throws error for unsupported platform (freebsd)', () => {
    Object.defineProperty(process, 'platform', { value: 'freebsd', configurable: true });
    Object.defineProperty(process, 'arch', { value: 'x64', configurable: true });

    expect(() => getPlatformAsset('1.0.0'))
      .toThrow('Unsupported platform: freebsd-x64');

    expect(() => getPlatformAsset('1.0.0'))
      .toThrow('Supported: linux/darwin/windows on amd64/arm64');
  });

  test('throws error for unsupported architecture (ia32)', () => {
    Object.defineProperty(process, 'platform', { value: 'linux', configurable: true });
    Object.defineProperty(process, 'arch', { value: 'ia32', configurable: true });

    expect(() => getPlatformAsset('1.0.0'))
      .toThrow('Unsupported platform: linux-ia32');

    expect(() => getPlatformAsset('1.0.0'))
      .toThrow('Supported: linux/darwin/windows on amd64/arm64');
  });
});

// SHA-256 of empty input
const EMPTY_SHA256 = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855';
// MD5 of empty input
const EMPTY_MD5 = 'd41d8cd98f00b204e9800998ecf8427e';

describe('verifyChecksum', () => {
  const mockReadFile = fs.promises.readFile as jest.MockedFunction<typeof fs.promises.readFile>;
  const mockCreateReadStream = fs.createReadStream as jest.MockedFunction<typeof fs.createReadStream>;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  // T-1143: releases publish SHA-256 sidecars; the action must verify with SHA-256.
  test('passes when SHA-256 checksums match', async () => {
    mockReadFile.mockResolvedValue(`${EMPTY_SHA256}\n`);

    // Mock stream that produces no data (SHA-256 of empty input)
    const mockStream: any = {
      on: jest.fn((event: string, handler: any) => {
        if (event === 'end') {
          handler();
        }
        return mockStream;
      })
    };
    mockCreateReadStream.mockReturnValue(mockStream);

    await expect(verifyChecksum('/path/to/file', '/path/to/checksum'))
      .resolves
      .toBeUndefined();
  });

  test('accepts sha256sum format with a trailing filename', async () => {
    mockReadFile.mockResolvedValue(`${EMPTY_SHA256}  rune-v1.0.0-linux-amd64.tar.gz\n`);
    const mockStream: any = {
      on: jest.fn((event: string, handler: any) => {
        if (event === 'end') handler();
        return mockStream;
      })
    };
    mockCreateReadStream.mockReturnValue(mockStream);

    await expect(verifyChecksum('/path/to/file', '/path/to/checksum'))
      .resolves
      .toBeUndefined();
  });

  test('rejects an MD5 digest even when it matches the archive', async () => {
    // MD5 of empty input; must not be accepted now that verification uses SHA-256
    mockReadFile.mockResolvedValue('d41d8cd98f00b204e9800998ecf8427e');
    const mockStream: any = {
      on: jest.fn((event: string, handler: any) => {
        if (event === 'end') handler();
        return mockStream;
      })
    };
    mockCreateReadStream.mockReturnValue(mockStream);

    await expect(verifyChecksum('/path/to/file', '/path/to/checksum'))
      .rejects
      .toThrow('SHA-256 checksum verification failed!');
  });

  test('throws error when checksums do not match', async () => {
    const expectedChecksum = 'a'.repeat(64);
    mockReadFile.mockResolvedValue(expectedChecksum);

    // Mock stream that produces a different SHA-256
    const mockStream: any = {
      on: jest.fn((event: string, handler: any) => {
        if (event === 'data') {
          handler(Buffer.from('test data'));
        } else if (event === 'end') {
          handler();
        }
        return mockStream;
      })
    };
    mockCreateReadStream.mockReturnValue(mockStream);

    await expect(verifyChecksum('/path/to/file', '/path/to/checksum'))
      .rejects
      .toThrow('SHA-256 checksum verification failed!');
  });

  test('throws error when checksum file cannot be read', async () => {
    const error = new Error('ENOENT: no such file or directory');
    mockReadFile.mockRejectedValue(error);

    await expect(verifyChecksum('/path/to/file', '/path/to/checksum'))
      .rejects
      .toThrow('ENOENT: no such file or directory');
  });

  test('throws error when archive file cannot be read', async () => {
    mockReadFile.mockResolvedValue(EMPTY_SHA256);

    const mockStream: any = {
      on: jest.fn((event: string, handler: any) => {
        if (event === 'error') {
          handler(new Error('ENOENT: archive not found'));
        }
        return mockStream;
      })
    };
    mockCreateReadStream.mockReturnValue(mockStream);

    await expect(verifyChecksum('/path/to/file', '/path/to/checksum'))
      .rejects
      .toThrow('ENOENT: archive not found');
  });
});

describe('installRune', () => {
  const mockCore = require('@actions/core');
  const mockTc = require('@actions/tool-cache');
  const mockExec = require('@actions/exec');
  const mockReadFile = fs.promises.readFile as jest.MockedFunction<typeof fs.promises.readFile>;
  const mockCreateReadStream = fs.createReadStream as jest.MockedFunction<typeof fs.createReadStream>;

  const mockOctokit = {
    rest: {
      repos: {
        getLatestRelease: jest.fn(),
        getReleaseByTag: jest.fn()
      }
    }
  };

  beforeEach(() => {
    jest.clearAllMocks();
    (getOctokit as jest.Mock).mockReturnValue(mockOctokit);

    // Set platform to linux/amd64 by default
    Object.defineProperty(process, 'platform', { value: 'linux', configurable: true });
    Object.defineProperty(process, 'arch', { value: 'x64', configurable: true });
  });

  test('returns cached version when cache hit', async () => {
    mockOctokit.rest.repos.getReleaseByTag.mockResolvedValue({
      data: { tag_name: 'v1.0.0' }
    });
    mockTc.find.mockReturnValue('/cache/rune/1.0.0/amd64');

    const result = await installRune('1.0.0', 'test-token');

    expect(result).toEqual({
      version: '1.0.0',
      path: '/cache/rune/1.0.0/amd64'
    });
    expect(mockCore.addPath).toHaveBeenCalledWith('/cache/rune/1.0.0/amd64');
    expect(mockTc.downloadTool).not.toHaveBeenCalled();
  });

  // T-2241: downloadTool's third argument is the full Authorization header value,
  // so the token must be sent with a scheme (e.g. "Bearer ghs_..."), not raw.
  test('sends a valid Authorization header for both downloads', async () => {
    mockOctokit.rest.repos.getReleaseByTag.mockResolvedValue({
      data: { tag_name: 'v1.0.0' }
    });
    mockTc.find.mockReturnValue('');
    mockTc.downloadTool
      .mockResolvedValueOnce('/tmp/archive')
      .mockResolvedValueOnce('/tmp/checksum');
    mockTc.extractTar.mockResolvedValue('/tmp/extracted');
    mockTc.cacheDir.mockResolvedValue('/cache/rune/1.0.0/amd64');
    mockReadFile.mockResolvedValue(EMPTY_SHA256);
    const mockStream: any = {
      on: jest.fn((event: string, handler: any) => {
        if (event === 'end') handler();
        return mockStream;
      })
    };
    mockCreateReadStream.mockReturnValue(mockStream);

    await installRune('1.0.0', 'ghs_example');

    expect(mockTc.downloadTool).toHaveBeenCalledTimes(2);
    for (const call of mockTc.downloadTool.mock.calls) {
      expect(call[2]).toBe('Bearer ghs_example');
    }
  });

  describe('checksum sidecar selection', () => {
    const base = 'https://github.com/ArjenSchwarz/rune/releases/download/v1.3.0/rune-v1.3.0-linux-amd64.tar.gz';

    beforeEach(() => {
      mockOctokit.rest.repos.getReleaseByTag.mockResolvedValue({ data: { tag_name: 'v1.3.0' } });
      mockTc.find.mockReturnValue('');
      mockTc.extractTar.mockResolvedValue('/tmp/extracted');
      mockTc.cacheDir.mockResolvedValue('/cache/rune/1.3.0/amd64');
      const mockStream: any = {
        on: jest.fn((event: string, handler: any) => {
          if (event === 'end') handler();
          return mockStream;
        })
      };
      mockCreateReadStream.mockReturnValue(mockStream);
    });

    const downloadedUrls = () => mockTc.downloadTool.mock.calls.map((c: any[]) => c[0]);

    test('prefers the SHA-256 sidecar when it exists', async () => {
      mockTc.downloadTool
        .mockResolvedValueOnce('/tmp/archive')
        .mockResolvedValueOnce('/tmp/checksum');
      mockReadFile.mockResolvedValue(EMPTY_SHA256);

      await installRune('1.3.0', 'ghs_example');

      expect(downloadedUrls()).toEqual([base, `${base}.sha256`]);
    });

    // Releases before v1.4.0 publish only .md5 sidecars
    test('falls back to the legacy MD5 sidecar when SHA-256 returns 404', async () => {
      mockTc.downloadTool
        .mockResolvedValueOnce('/tmp/archive')
        .mockRejectedValueOnce(new HTTPError(404))
        .mockResolvedValueOnce('/tmp/checksum.md5');
      mockReadFile.mockResolvedValue(EMPTY_MD5);

      const result = await installRune('1.3.0', 'ghs_example');

      expect(downloadedUrls()).toEqual([base, `${base}.sha256`, `${base}.md5`]);
      for (const call of mockTc.downloadTool.mock.calls) {
        expect(call[2]).toBe('Bearer ghs_example');
      }
      expect(mockReadFile).toHaveBeenCalledWith('/tmp/checksum.md5', 'utf8');
      expect(result.path).toBe('/cache/rune/1.3.0/amd64');
    });

    test('legacy MD5 mismatch is fatal', async () => {
      mockTc.downloadTool
        .mockResolvedValueOnce('/tmp/archive')
        .mockRejectedValueOnce(new HTTPError(404))
        .mockResolvedValueOnce('/tmp/checksum.md5');
      mockReadFile.mockResolvedValue('0'.repeat(32));

      await expect(installRune('1.3.0', 'ghs_example'))
        .rejects.toThrow('MD5 checksum verification failed!');
      expect(mockTc.extractTar).not.toHaveBeenCalled();
    });

    test('SHA-256 mismatch is fatal and does not fall back to MD5', async () => {
      mockTc.downloadTool
        .mockResolvedValueOnce('/tmp/archive')
        .mockResolvedValueOnce('/tmp/checksum');
      mockReadFile.mockResolvedValue('a'.repeat(64));

      await expect(installRune('1.3.0', 'ghs_example'))
        .rejects.toThrow('SHA-256 checksum verification failed!');
      expect(downloadedUrls()).toEqual([base, `${base}.sha256`]);
      expect(mockTc.extractTar).not.toHaveBeenCalled();
    });

    test.each([
      ['401 Unauthorized', new HTTPError(401)],
      ['403 Forbidden', new HTTPError(403)],
      ['500 Server Error', new HTTPError(500)],
      ['network error', new Error('ECONNRESET')]
    ])('SHA-256 sidecar download failure (%s) is fatal and does not fall back', async (_name, error) => {
      mockTc.downloadTool
        .mockResolvedValueOnce('/tmp/archive')
        .mockRejectedValueOnce(error);

      await expect(installRune('1.3.0', 'ghs_example')).rejects.toBe(error);
      expect(downloadedUrls()).toEqual([base, `${base}.sha256`]);
      expect(mockTc.extractTar).not.toHaveBeenCalled();
    });
  });

  test('downloads and caches on cache miss for Linux', async () => {
    mockOctokit.rest.repos.getReleaseByTag.mockResolvedValue({
      data: { tag_name: 'v1.0.0' }
    });
    mockTc.find.mockReturnValue('');
    mockTc.downloadTool
      .mockResolvedValueOnce('/tmp/archive')
      .mockResolvedValueOnce('/tmp/checksum');
    mockTc.extractTar.mockResolvedValue('/tmp/extracted');
    mockTc.cacheDir.mockResolvedValue('/cache/rune/1.0.0/amd64');

    // Mock checksum verification
    mockReadFile.mockResolvedValue(EMPTY_SHA256);
    const mockStream: any = {
      on: jest.fn((event: string, handler: any) => {
        if (event === 'end') handler();
        return mockStream;
      })
    };
    mockCreateReadStream.mockReturnValue(mockStream);

    const result = await installRune('1.0.0', 'test-token');

    expect(mockTc.downloadTool).toHaveBeenCalledWith(
      'https://github.com/ArjenSchwarz/rune/releases/download/v1.0.0/rune-v1.0.0-linux-amd64.tar.gz',
      undefined,
      'Bearer test-token'
    );
    expect(mockTc.downloadTool).toHaveBeenCalledWith(
      'https://github.com/ArjenSchwarz/rune/releases/download/v1.0.0/rune-v1.0.0-linux-amd64.tar.gz.sha256',
      undefined,
      'Bearer test-token'
    );
    expect(mockTc.extractTar).toHaveBeenCalledWith('/tmp/archive');
    expect(mockExec.exec).toHaveBeenCalledWith('chmod', ['+x', '/tmp/extracted/rune']);
    expect(mockTc.cacheDir).toHaveBeenCalledWith('/tmp/extracted', 'rune', '1.0.0', 'amd64');
    expect(result).toEqual({
      version: '1.0.0',
      path: '/cache/rune/1.0.0/amd64'
    });
  });

  test('downloads and caches for Windows with zip extraction', async () => {
    Object.defineProperty(process, 'platform', { value: 'win32', configurable: true });

    mockOctokit.rest.repos.getReleaseByTag.mockResolvedValue({
      data: { tag_name: 'v1.0.0' }
    });
    mockTc.find.mockReturnValue('');
    mockTc.downloadTool
      .mockResolvedValueOnce('/tmp/archive')
      .mockResolvedValueOnce('/tmp/checksum');
    mockTc.extractZip.mockResolvedValue('/tmp/extracted');
    mockTc.cacheDir.mockResolvedValue('/cache/rune/1.0.0/amd64');

    // Mock checksum verification
    mockReadFile.mockResolvedValue(EMPTY_SHA256);
    const mockStream: any = {
      on: jest.fn((event: string, handler: any) => {
        if (event === 'end') handler();
        return mockStream;
      })
    };
    mockCreateReadStream.mockReturnValue(mockStream);

    const result = await installRune('1.0.0', 'test-token');

    expect(mockTc.downloadTool).toHaveBeenCalledWith(
      'https://github.com/ArjenSchwarz/rune/releases/download/v1.0.0/rune-v1.0.0-windows-amd64.zip',
      undefined,
      'Bearer test-token'
    );
    expect(mockTc.extractZip).toHaveBeenCalledWith('/tmp/archive');
    expect(mockExec.exec).not.toHaveBeenCalled(); // No chmod on Windows
    expect(result).toEqual({
      version: '1.0.0',
      path: '/cache/rune/1.0.0/amd64'
    });
  });

  test('uses arm64 architecture when detected', async () => {
    Object.defineProperty(process, 'arch', { value: 'arm64', configurable: true });

    mockOctokit.rest.repos.getReleaseByTag.mockResolvedValue({
      data: { tag_name: 'v1.0.0' }
    });
    mockTc.find.mockReturnValue('');
    mockTc.downloadTool
      .mockResolvedValueOnce('/tmp/archive')
      .mockResolvedValueOnce('/tmp/checksum');
    mockTc.extractTar.mockResolvedValue('/tmp/extracted');
    mockTc.cacheDir.mockResolvedValue('/cache/rune/1.0.0/arm64');

    // Mock checksum verification
    mockReadFile.mockResolvedValue(EMPTY_SHA256);
    const mockStream: any = {
      on: jest.fn((event: string, handler: any) => {
        if (event === 'end') handler();
        return mockStream;
      })
    };
    mockCreateReadStream.mockReturnValue(mockStream);

    const result = await installRune('1.0.0', 'test-token');

    expect(mockTc.find).toHaveBeenCalledWith('rune', '1.0.0', 'arm64');
    expect(mockTc.downloadTool).toHaveBeenCalledWith(
      'https://github.com/ArjenSchwarz/rune/releases/download/v1.0.0/rune-v1.0.0-linux-arm64.tar.gz',
      undefined,
      'Bearer test-token'
    );
    expect(mockTc.cacheDir).toHaveBeenCalledWith('/tmp/extracted', 'rune', '1.0.0', 'arm64');
    expect(result).toEqual({
      version: '1.0.0',
      path: '/cache/rune/1.0.0/arm64'
    });
  });

  test('resolves latest version before downloading', async () => {
    mockOctokit.rest.repos.getLatestRelease.mockResolvedValue({
      data: { tag_name: 'v2.5.3' }
    });
    mockTc.find.mockReturnValue('');
    mockTc.downloadTool
      .mockResolvedValueOnce('/tmp/archive')
      .mockResolvedValueOnce('/tmp/checksum');
    mockTc.extractTar.mockResolvedValue('/tmp/extracted');
    mockTc.cacheDir.mockResolvedValue('/cache/rune/2.5.3/amd64');

    // Mock checksum verification
    mockReadFile.mockResolvedValue(EMPTY_SHA256);
    const mockStream: any = {
      on: jest.fn((event: string, handler: any) => {
        if (event === 'end') handler();
        return mockStream;
      })
    };
    mockCreateReadStream.mockReturnValue(mockStream);

    const result = await installRune('latest', 'test-token');

    expect(mockOctokit.rest.repos.getLatestRelease).toHaveBeenCalled();
    expect(mockTc.downloadTool).toHaveBeenCalledWith(
      'https://github.com/ArjenSchwarz/rune/releases/download/v2.5.3/rune-v2.5.3-linux-amd64.tar.gz',
      undefined,
      'Bearer test-token'
    );
    expect(result.version).toBe('2.5.3');
  });
});
