[![Build Status](https://github.com/myopenfactory/edi-connector/workflows/CI/badge.svg)](https://github.com/myopenfactory/edi-connector/actions?query=workflow%3ACI)

# EDI-Connector

![EDI-Connector Logo](doc/EDI-Connector.png)

## Overview

EDI-Connector is a custom software solution provided by myOpenFactory Software GmbH that facilitates seamless integration with the EDI-Platform. It enables customers to send and receive EDI messages efficiently, ensuring smooth and simple electronic data interchange.

## Usage

The EDI-Connector serves as a bridge between the customer's system and the myOpenFactory EDI-Platform, simplifying electronic business communication.

See [Documentation](https://docs.myopenfactory.com/edi/protocols/edi-connector/) for more information.

## Windows Installer and SmartScreen Warnings

The Windows installer (`edi-connector-install.exe`) is signed with an EV code signing certificate issued to **myOpenFactory Software GmbH**. Since the EDI-Connector is only distributed to a small number of customers, Microsoft Defender SmartScreen does not have enough download statistics for it and Microsoft Edge may warn that the file *"isn't commonly downloaded"*. This is a reputation warning, not a malware detection.

> **Note:** Installers attached to pre-releases (`-rc` versions) are **not signed**. Use regular releases for production installations.

### Install via WinGet (recommended)

Installing with [WinGet](https://learn.microsoft.com/en-us/windows/package-manager/winget/) avoids the browser download and therefore the SmartScreen warning. Run in an elevated terminal:

```powershell
winget install --id myOpenFactory.EDIConnector --exact
```

Updates can be installed with:

```powershell
winget upgrade --id myOpenFactory.EDIConnector --exact
```

New versions are submitted to the WinGet repository automatically on every release. They are available after Microsoft's review of the submission, which usually takes a few hours up to a few days.

### Verify the signature

Before bypassing the warning, make sure the file is genuine:

1. Right-click `edi-connector-install.exe` → **Properties** → **Digital Signatures**.
2. The signer must be **myOpenFactory Software GmbH** and **Details** must report *"This digital signature is OK."*

Or use PowerShell:

```powershell
Get-AuthenticodeSignature .\edi-connector-install.exe | Format-List Status, SignerCertificate
```

`Status` must be `Valid` and the certificate subject must contain `O=myOpenFactory Software GmbH`.

### Download and run the installer (end users)

1. In the Edge download list, hover over the blocked download, click **…** → **Keep**.
2. Click **Show more** → **Keep anyway**.
3. If Windows shows *"Windows protected your PC"* when starting the installer, click **More info**, check that the publisher is *myOpenFactory Software GmbH* and click **Run anyway**.

### Allow the download company-wide (IT administrators)

To avoid the warning for all users, administrators can exempt the GitHub release download domains for `.exe` files with the Edge policy [`ExemptSmartScreenDownloadWarnings`](https://learn.microsoft.com/en-us/deployedge/microsoft-edge-browser-policies/exemptsmartscreendownloadwarnings) (via Group Policy, Intune or registry):

```json
[{"file_extension": "exe", "domains": ["https://github.com", "https://objects.githubusercontent.com", "https://release-assets.githubusercontent.com"]}]
```

Registry example:

```
reg add "HKLM\SOFTWARE\Policies\Microsoft\Edge" /v ExemptSmartScreenDownloadWarnings /t REG_SZ /d "[{\"file_extension\": \"exe\", \"domains\": [\"https://github.com\", \"https://objects.githubusercontent.com\", \"https://release-assets.githubusercontent.com\"]}]" /f
```

Organizations using Microsoft Defender for Endpoint can additionally add an **Allow** indicator for the myOpenFactory signing certificate (Microsoft Defender portal → **Settings** → **Endpoints** → **Indicators** → **Certificates**). Export the certificate from the installer's **Digital Signatures** tab (**Details** → **View Certificate** → **Details** → **Copy to File…**) and upload it there.
