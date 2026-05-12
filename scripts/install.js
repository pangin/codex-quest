#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const zlib = require('zlib');
const tar = require('tar');

const REPO = 'pangin/codex-quest';
const BINARY_NAME = process.platform === 'win32' ? 'cxq-bin.exe' : 'cxq-bin';

function getPlatformInfo() {
  const platform = process.platform;
  const arch = process.arch;

  const platformMap = {
    darwin: 'darwin',
    linux: 'linux',
    win32: 'windows',
  };

  const archMap = {
    x64: 'amd64',
    arm64: 'arm64',
  };

  const os = platformMap[platform];
  const cpu = archMap[arch];

  if (!os || !cpu) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }

  // Windows arm64 not supported
  if (os === 'windows' && cpu === 'arm64') {
    throw new Error('Windows ARM64 is not supported yet');
  }

  // Linux arm64 not supported yet
  if (os === 'linux' && cpu === 'arm64') {
    throw new Error('Linux ARM64 is not supported yet');
  }

  return { os, cpu, isWindows: platform === 'win32' };
}

function getAssetName(version, os, cpu) {
  const ext = os === 'windows' ? 'zip' : 'tar.gz';
  return `codex-quest_${version}_${os}_${cpu}.${ext}`;
}

async function getLatestRelease() {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: 'api.github.com',
      path: `/repos/${REPO}/releases/latest`,
      headers: {
        'User-Agent': 'codex-quest-npm-installer',
      },
    };

    https.get(options, (res) => {
      let data = '';
      res.on('data', (chunk) => (data += chunk));
      res.on('end', () => {
        try {
          resolve(JSON.parse(data));
        } catch (e) {
          reject(e);
        }
      });
    }).on('error', reject);
  });
}

async function downloadFile(url, destPath) {
  return new Promise((resolve, reject) => {
    const maxRetries = 3;
    let retryCount = 0;

    const attempt = () => {
      const file = fs.createWriteStream(destPath);

      const fetch = (url) => {
        const req = https.get(url, {
          headers: { 'User-Agent': 'codex-quest-npm-installer' },
          timeout: 30000,
        }, (response) => {
          if (response.statusCode === 302 || response.statusCode === 301) {
            fetch(response.headers.location);
            return;
          }

          if (response.statusCode !== 200) {
            file.destroy();
            const err = new Error(`Failed to download: ${response.statusCode}`);
            if (retryCount < maxRetries) {
              retryCount++;
              console.log(`Retry ${retryCount}/${maxRetries}...`);
              setTimeout(attempt, 1000);
            } else {
              reject(err);
            }
            return;
          }

          response.pipe(file);
          file.on('finish', () => {
            file.close(resolve);
          });
        }).on('error', (err) => {
          file.destroy();
          fs.unlink(destPath, () => {});
          if (retryCount < maxRetries) {
            retryCount++;
            console.log(`Network error, retry ${retryCount}/${maxRetries}...`);
            setTimeout(attempt, 1000);
          } else {
            reject(err);
          }
        });
        req.setTimeout(30000);
      };

      fetch(url);
    };

    attempt();
  });
}

async function extractTarGz(archivePath, destDir, binaryName) {
  return new Promise((resolve, reject) => {
    fs.createReadStream(archivePath)
      .pipe(zlib.createGunzip())
      .pipe(tar.extract({ cwd: destDir }))
      .on('finish', resolve)
      .on('error', reject);
  });
}

async function extractZip(archivePath, destDir) {
  // Use system unzip on Unix, PowerShell on Windows
  if (process.platform === 'win32') {
    execSync(`powershell -command "Expand-Archive -Path '${archivePath}' -DestinationPath '${destDir}' -Force"`);
  } else {
    execSync(`unzip -o "${archivePath}" -d "${destDir}"`);
  }
}

async function main() {
  try {
    console.log('Installing Codex Quest...');

    const { os, cpu, isWindows } = getPlatformInfo();
    console.log(`Platform: ${os}-${cpu}`);

    // Get latest release
    const release = await getLatestRelease();
    const version = release.tag_name;
    console.log(`Latest version: ${version}`);

    // Find the right asset
    const assetName = getAssetName(version, os, cpu);
    const asset = release.assets.find((a) => a.name === assetName);

    if (!asset) {
      throw new Error(`Could not find release asset: ${assetName}`);
    }

    // Download
    const binDir = path.join(__dirname, '..', 'bin');
    const tmpDir = path.join(__dirname, '..', '.tmp');

    if (!fs.existsSync(binDir)) fs.mkdirSync(binDir, { recursive: true });
    if (!fs.existsSync(tmpDir)) fs.mkdirSync(tmpDir, { recursive: true });

    const archivePath = path.join(tmpDir, assetName);
    console.log(`Downloading ${assetName}...`);
    await downloadFile(asset.browser_download_url, archivePath);

    // Extract
    console.log('Extracting...');
    const destBinary = path.join(binDir, BINARY_NAME);
    if (isWindows) {
      await extractZip(archivePath, tmpDir);
      fs.renameSync(
        path.join(tmpDir, `cxq-windows-amd64.exe`),
        destBinary
      );
    } else {
      await extractTarGz(archivePath, tmpDir);
      fs.renameSync(path.join(tmpDir, `cxq-${os}-${cpu}`), destBinary);
      fs.chmodSync(destBinary, '755');
    }

    // Cleanup - with retry logic
    console.log('Cleaning up...');
    let cleanupAttempts = 0;
    while (cleanupAttempts < 3) {
      try {
        // Remove any remaining files first
        if (fs.existsSync(archivePath)) {
          fs.unlinkSync(archivePath);
        }
        // Then remove directory
        if (fs.existsSync(tmpDir)) {
          fs.rmSync(tmpDir, { recursive: true, force: true });
        }
        break;
      } catch (err) {
        cleanupAttempts++;
        if (cleanupAttempts >= 3) {
          console.warn('Warning: Could not clean up temporary files:', err.message);
        } else {
          await new Promise(resolve => setTimeout(resolve, 100));
        }
      }
    }

    console.log('Codex Quest installed successfully!');
    console.log('Run "cxq" to watch your current project.');
  } catch (error) {
    console.error('Installation failed:', error.message);
    process.exit(1);
  }
}

main();
