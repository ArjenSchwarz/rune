import * as core from '@actions/core';
import * as tc from '@actions/tool-cache';
import * as exec from '@actions/exec';
import * as crypto from 'crypto';
import * as fs from 'fs';
import * as path from 'path';
import { getOctokit } from '@actions/github';

export async function resolveVersion(version: string, token: string): Promise<string> {
  const octokit = getOctokit(token);
  const normalized = version.replace(/^v/, '');

  if (normalized === 'latest') {
    const { data } = await octokit.rest.repos.getLatestRelease({
      owner: 'ArjenSchwarz',
      repo: 'rune'
    });
    return data.tag_name.replace(/^v/, '');
  }

  // Verify exact version exists
  try {
    await octokit.rest.repos.getReleaseByTag({
      owner: 'ArjenSchwarz',
      repo: 'rune',
      tag: `v${normalized}`
    });
    return normalized;
  } catch (error: any) {
    if (error.status === 404) {
      throw new Error(
        `Version ${version} not found.\n` +
        `Check available versions at: https://github.com/ArjenSchwarz/rune/releases`
      );
    }
    throw error;
  }
}

export function getPlatformAsset(version: string): { assetName: string; isWindows: boolean } {
  const osMap: Record<string, string> = {
    'linux': 'linux',
    'darwin': 'darwin',
    'win32': 'windows'
  };

  const archMap: Record<string, string> = {
    'x64': 'amd64',
    'arm64': 'arm64'
  };

  const os = osMap[process.platform];
  const arch = archMap[process.arch];

  if (!os || !arch) {
    throw new Error(
      `Unsupported platform: ${process.platform}-${process.arch}\n` +
      `Supported: linux/darwin/windows on amd64/arm64`
    );
  }

  const ext = os === 'windows' ? 'zip' : 'tar.gz';
  const assetName = `rune-v${version}-${os}-${arch}.${ext}`;

  return { assetName, isWindows: os === 'windows' };
}

export async function verifyChecksum(filePath: string, checksumPath: string): Promise<void> {
  const expected = (await fs.promises.readFile(checksumPath, 'utf8')).trim();

  // Calculate MD5
  const hash = crypto.createHash('md5');
  const stream = fs.createReadStream(filePath);

  await new Promise<void>((resolve, reject) => {
    stream.on('data', data => hash.update(data));
    stream.on('end', () => resolve());
    stream.on('error', reject);
  });

  const actual = hash.digest('hex');

  if (expected !== actual) {
    throw new Error(
      `MD5 checksum verification failed!\n` +
      `Expected: ${expected}\n` +
      `Actual: ${actual}`
    );
  }

  core.info('✓ Checksum verified');
}

export async function installRune(
  version: string,
  token: string
): Promise<{ version: string; path: string }> {

  // 1. Resolve version FIRST
  const resolvedVersion = await resolveVersion(version, token);

  // 2. Get platform info and asset name
  const { assetName, isWindows } = getPlatformAsset(resolvedVersion);
  const arch = process.arch === 'arm64' ? 'arm64' : 'amd64';

  // 3. Check cache
  const cachedPath = tc.find('rune', resolvedVersion, arch);
  if (cachedPath) {
    core.info(`Using cached rune ${resolvedVersion}`);
    core.addPath(cachedPath);
    return { version: resolvedVersion, path: cachedPath };
  }

  // 4. Download
  const baseUrl = `https://github.com/ArjenSchwarz/rune/releases/download/v${resolvedVersion}`;
  core.info(`Downloading rune ${resolvedVersion}...`);

  const archivePath = await tc.downloadTool(`${baseUrl}/${assetName}`, undefined, token);
  const checksumPath = await tc.downloadTool(`${baseUrl}/${assetName}.md5`, undefined, token);

  // 5. Verify checksum
  await verifyChecksum(archivePath, checksumPath);

  // 6. Extract
  const extractedPath = isWindows
    ? await tc.extractZip(archivePath)
    : await tc.extractTar(archivePath);

  // 7. Make executable (non-Windows)
  if (!isWindows) {
    const binaryPath = path.join(extractedPath, 'rune');
    await exec.exec('chmod', ['+x', binaryPath]);
  }

  // 8. Cache and add to PATH
  const cachedToolPath = await tc.cacheDir(extractedPath, 'rune', resolvedVersion, arch);
  core.addPath(cachedToolPath);

  core.info(`✓ Successfully installed rune ${resolvedVersion}`);
  return { version: resolvedVersion, path: cachedToolPath };
}
