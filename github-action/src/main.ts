import * as core from '@actions/core';
import { installRune } from './install';

async function run(): Promise<void> {
  try {
    const version = core.getInput('version') || 'latest';
    const token = core.getInput('github-token') || process.env.GITHUB_TOKEN;

    const result = await installRune(version, token);

    core.setOutput('version', result.version);
    core.setOutput('path', result.path);

    core.info(`✓ rune ${result.version} installed successfully`);
  } catch (error) {
    core.setFailed(error instanceof Error ? error.message : String(error));
  }
}

run();
