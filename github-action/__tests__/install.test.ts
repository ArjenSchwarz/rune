import { resolveVersion, getPlatformAsset, verifyChecksum, installRune } from '../src/install';
import { getOctokit } from '@actions/github';
import * as fs from 'fs';

// Mock dependencies
jest.mock('@actions/github');
jest.mock('@actions/core');
jest.mock('@actions/tool-cache');
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

describe('verifyChecksum', () => {
  const mockReadFile = fs.promises.readFile as jest.MockedFunction<typeof fs.promises.readFile>;
  const mockCreateReadStream = fs.createReadStream as jest.MockedFunction<typeof fs.createReadStream>;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('passes when checksums match', async () => {
    const expectedChecksum = 'd41d8cd98f00b204e9800998ecf8427e';
    mockReadFile.mockResolvedValue(expectedChecksum);

    // Mock stream that produces the expected MD5
    const mockStream: any = {
      on: jest.fn((event: string, handler: any) => {
        if (event === 'data') {
          // Empty data produces MD5: d41d8cd98f00b204e9800998ecf8427e
        } else if (event === 'end') {
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

  test('throws error when checksums do not match', async () => {
    const expectedChecksum = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
    mockReadFile.mockResolvedValue(expectedChecksum);

    // Mock stream that produces different MD5
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
      .toThrow('MD5 checksum verification failed!');
  });

  test('throws error when checksum file cannot be read', async () => {
    const error = new Error('ENOENT: no such file or directory');
    mockReadFile.mockRejectedValue(error);

    await expect(verifyChecksum('/path/to/file', '/path/to/checksum'))
      .rejects
      .toThrow('ENOENT: no such file or directory');
  });

  test('throws error when archive file cannot be read', async () => {
    mockReadFile.mockResolvedValue('d41d8cd98f00b204e9800998ecf8427e');

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
    mockReadFile.mockResolvedValue('d41d8cd98f00b204e9800998ecf8427e');
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
      'test-token'
    );
    expect(mockTc.downloadTool).toHaveBeenCalledWith(
      'https://github.com/ArjenSchwarz/rune/releases/download/v1.0.0/rune-v1.0.0-linux-amd64.tar.gz.md5',
      undefined,
      'test-token'
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
    mockReadFile.mockResolvedValue('d41d8cd98f00b204e9800998ecf8427e');
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
      'test-token'
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
    mockReadFile.mockResolvedValue('d41d8cd98f00b204e9800998ecf8427e');
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
      'test-token'
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
    mockReadFile.mockResolvedValue('d41d8cd98f00b204e9800998ecf8427e');
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
      'test-token'
    );
    expect(result.version).toBe('2.5.3');
  });
});
