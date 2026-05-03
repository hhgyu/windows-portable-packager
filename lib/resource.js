"use strict";

const fs = require("fs");
const path = require("path");

const RT_ICON = 3;
const RT_GROUP_ICON = 14;
const RT_VERSION = 16;

const MACHINE = { amd64: 0x8664, "386": 0x014c, arm64: 0xaa64 };
const RELOC_ADDR32NB = { amd64: 0x0003, "386": 0x0007, arm64: 0x0002 };

function resolveIconPath(projectDir, pkg) {
  const candidates = [];
  const winIcon = pkg.build && pkg.build.win && pkg.build.win.icon;
  if (winIcon) candidates.push(path.resolve(projectDir, winIcon));
  const buildIcon = pkg.build && pkg.build.icon;
  if (buildIcon) {
    if (!buildIcon.endsWith(".ico")) candidates.push(path.resolve(projectDir, buildIcon + ".ico"));
    candidates.push(path.resolve(projectDir, buildIcon));
  }
  candidates.push(path.join(projectDir, "build", "icon.ico"));
  candidates.push(path.join(projectDir, "icon.ico"));
  return candidates.find(fs.existsSync) || null;
}

function parseAuthor(author) {
  if (!author) return "";
  if (typeof author === "string") return author.replace(/<[^>]+>/g, "").replace(/\([^)]+\)/g, "").trim();
  return author.name || "";
}

function versionToWords(version) {
  const parts = String(version || "0.0.0").replace(/[^0-9.]/g, "").split(".").map(Number);
  while (parts.length < 4) parts.push(0);
  return parts.map(n => Math.min(n, 65535));
}

function parseIco(buf) {
  const count = buf.readUInt16LE(4);
  const images = [];
  for (let i = 0; i < count; i++) {
    const off = 6 + i * 16;
    const dataSize = buf.readUInt32LE(off + 8);
    const dataOffset = buf.readUInt32LE(off + 12);
    images.push({
      width: buf[off] || 256,
      height: buf[off + 1] || 256,
      colorCount: buf[off + 2],
      planes: buf.readUInt16LE(off + 4),
      bitCount: buf.readUInt16LE(off + 6),
      data: buf.slice(dataOffset, dataOffset + dataSize),
    });
  }
  return images;
}

function buildGroupIconData(images) {
  const buf = Buffer.alloc(6 + images.length * 14, 0);
  buf.writeUInt16LE(0, 0);
  buf.writeUInt16LE(1, 2);
  buf.writeUInt16LE(images.length, 4);
  images.forEach((img, i) => {
    const off = 6 + i * 14;
    buf[off] = img.width === 256 ? 0 : img.width;
    buf[off + 1] = img.height === 256 ? 0 : img.height;
    buf[off + 2] = img.colorCount;
    buf[off + 3] = 0;
    buf.writeUInt16LE(img.planes, off + 4);
    buf.writeUInt16LE(img.bitCount, off + 6);
    buf.writeUInt32LE(img.data.length, off + 8);
    buf.writeUInt16LE(i + 1, off + 12);
  });
  return buf;
}

function utf16le(str) {
  const buf = Buffer.alloc((str.length + 1) * 2, 0);
  for (let i = 0; i < str.length; i++) buf.writeUInt16LE(str.charCodeAt(i), i * 2);
  return buf;
}

function align4(n) { return (n + 3) & ~3; }

function writeVersionBlock(name, value, children) {
  const nameBuf = utf16le(name);
  const isText = value !== null && value !== undefined;
  const valueLen = isText ? value.length : 0;

  const headerBase = 6 + nameBuf.length;
  const headerPadded = align4(headerBase);
  const valuePadded = isText ? align4(headerPadded + valueLen) : headerPadded;

  const childrenBuf = (children && children.length) ? Buffer.concat(children) : Buffer.alloc(0);
  const totalLen = valuePadded + childrenBuf.length;
  const buf = Buffer.alloc(totalLen, 0);

  buf.writeUInt16LE(totalLen, 0);
  buf.writeUInt16LE(isText ? value.length / 2 : 0, 2);
  buf.writeUInt16LE(isText ? 1 : 0, 4);
  nameBuf.copy(buf, 6);
  if (isText) value.copy(buf, headerPadded);
  if (childrenBuf.length) childrenBuf.copy(buf, valuePadded);

  return buf;
}

function buildVersionData(pkg) {
  const build = pkg.build || {};
  const win = build.win || {};
  const productName = build.productName || pkg.name || "";
  const ver = versionToWords(pkg.version);
  const companyName = win.companyName || parseAuthor(pkg.author) || "";
  const copyright = build.copyright || (companyName ? `\u00A9 ${new Date().getFullYear()} ${companyName}` : "");
  const versionStr = pkg.version || "0.0.0";

  const strings = {
    CompanyName: companyName,
    FileDescription: pkg.description || productName,
    FileVersion: versionStr,
    InternalName: productName,
    LegalCopyright: copyright,
    OriginalFilename: productName + ".exe",
    ProductName: productName,
    ProductVersion: versionStr,
  };

  const stringBlocks = Object.entries(strings).map(([k, v]) =>
    writeVersionBlock(k, utf16le(v), null)
  );
  const stringTable = writeVersionBlock("040904B0", null, stringBlocks);
  const stringFileInfo = writeVersionBlock("StringFileInfo", null, [stringTable]);

  const varValue = Buffer.alloc(4);
  varValue.writeUInt16LE(0x0409, 0);
  varValue.writeUInt16LE(0x04B0, 2);
  const varBlock = writeVersionBlock("Translation", varValue, null);
  const varFileInfo = writeVersionBlock("VarFileInfo", null, [varBlock]);

  const fixedInfo = Buffer.alloc(52, 0);
  fixedInfo.writeUInt32LE(0xFEEF04BD, 0);
  fixedInfo.writeUInt32LE(0x00010000, 4);
  fixedInfo.writeUInt16LE(ver[1], 8);  fixedInfo.writeUInt16LE(ver[0], 10);
  fixedInfo.writeUInt16LE(ver[3], 12); fixedInfo.writeUInt16LE(ver[2], 14);
  fixedInfo.writeUInt16LE(ver[1], 16); fixedInfo.writeUInt16LE(ver[0], 18);
  fixedInfo.writeUInt16LE(ver[3], 20); fixedInfo.writeUInt16LE(ver[2], 22);
  fixedInfo.writeUInt32LE(0x3F, 24);
  fixedInfo.writeUInt32LE(0x00, 28);
  fixedInfo.writeUInt32LE(0x00040004, 32);
  fixedInfo.writeUInt32LE(0x00000001, 36);

  return writeVersionBlock("VS_VERSION_INFO", fixedInfo, [stringFileInfo, varFileInfo]);
}

function buildCoff(resources, goArch) {
  const machine = MACHINE[goArch] || MACHINE.amd64;
  const relocType = RELOC_ADDR32NB[goArch] || RELOC_ADDR32NB.amd64;

  const typeMap = new Map();
  for (const res of resources) {
    if (!typeMap.has(res.type)) typeMap.set(res.type, []);
    typeMap.get(res.type).push(res);
  }
  for (const list of typeMap.values()) list.sort((a, b) => a.id - b.id);

  const types = [...typeMap.keys()].sort((a, b) => a - b);
  const orderedResources = types.flatMap(t => typeMap.get(t));

  const DIR_HEADER = 16;
  const DIR_ENTRY = 8;
  const DATA_ENTRY = 16;

  const numTypes = types.length;
  const totalIds = orderedResources.length;

  const lvl1Size = DIR_HEADER + numTypes * DIR_ENTRY;
  let lvl2Size = 0;
  for (const t of types) lvl2Size += DIR_HEADER + typeMap.get(t).length * DIR_ENTRY;
  const lvl3Size = totalIds * (DIR_HEADER + DIR_ENTRY);
  const dirSize = lvl1Size + lvl2Size + lvl3Size;
  const dataEntriesOff = dirSize;

  let rawOff = dataEntriesOff + totalIds * DATA_ENTRY;
  const dataOffsets = [];
  for (const res of orderedResources) {
    dataOffsets.push(rawOff);
    rawOff += res.data.length;
    if (rawOff % 4 !== 0) rawOff += 4 - (rawOff % 4);
  }
  const rsrcSize = rawOff;

  const rsrcBuf = Buffer.alloc(rsrcSize, 0);
  const relocOffsets = [];

  rsrcBuf.writeUInt16LE(0, 12);
  rsrcBuf.writeUInt16LE(numTypes, 14);

  let lvl2Off = lvl1Size;
  let lvl3Off = lvl1Size + lvl2Size;
  let dataIdx = 0;

  for (let ti = 0; ti < types.length; ti++) {
    const typeRes = typeMap.get(types[ti]);
    rsrcBuf.writeUInt32LE(types[ti], DIR_HEADER + ti * DIR_ENTRY);
    rsrcBuf.writeUInt32LE((0x80000000 | lvl2Off) >>> 0, DIR_HEADER + ti * DIR_ENTRY + 4);

    rsrcBuf.writeUInt16LE(0, lvl2Off + 12);
    rsrcBuf.writeUInt16LE(typeRes.length, lvl2Off + 14);

    for (let ri = 0; ri < typeRes.length; ri++) {
      const lvl2EntryOff = lvl2Off + DIR_HEADER + ri * DIR_ENTRY;
      rsrcBuf.writeUInt32LE(typeRes[ri].id, lvl2EntryOff);
      rsrcBuf.writeUInt32LE((0x80000000 | lvl3Off) >>> 0, lvl2EntryOff + 4);

      rsrcBuf.writeUInt16LE(0, lvl3Off + 12);
      rsrcBuf.writeUInt16LE(1, lvl3Off + 14);

      const lvl3EntryOff = lvl3Off + DIR_HEADER;
      rsrcBuf.writeUInt32LE(0x0409, lvl3EntryOff);
      rsrcBuf.writeUInt32LE(dataEntriesOff + dataIdx * DATA_ENTRY, lvl3EntryOff + 4);

      const deOff = dataEntriesOff + dataIdx * DATA_ENTRY;
      rsrcBuf.writeUInt32LE(dataOffsets[dataIdx], deOff);
      relocOffsets.push(deOff);
      rsrcBuf.writeUInt32LE(orderedResources[dataIdx].data.length, deOff + 4);

      lvl3Off += DIR_HEADER + DIR_ENTRY;
      dataIdx++;
    }
    lvl2Off += DIR_HEADER + typeRes.length * DIR_ENTRY;
  }

  let writeOff = dataEntriesOff + totalIds * DATA_ENTRY;
  for (const res of orderedResources) {
    res.data.copy(rsrcBuf, writeOff);
    writeOff += res.data.length;
    if (writeOff % 4 !== 0) writeOff += 4 - (writeOff % 4);
  }

  const numRelocs = relocOffsets.length;
  const COFF_HEADER = 20;
  const SECTION_HEADER = 40;
  const SYMBOL_SIZE = 18;
  const RELOC_SIZE = 10;

  const sectionDataOff = COFF_HEADER + SECTION_HEADER;
  const relocTableOff = sectionDataOff + rsrcSize;
  const symbolTableOff = relocTableOff + numRelocs * RELOC_SIZE;
  const stringTableOff = symbolTableOff + SYMBOL_SIZE;
  const totalSize = stringTableOff + 4;

  const buf = Buffer.alloc(totalSize, 0);

  buf.writeUInt16LE(machine, 0);
  buf.writeUInt16LE(1, 2);
  buf.writeUInt32LE(0, 4);
  buf.writeUInt32LE(symbolTableOff, 12);
  buf.writeUInt32LE(1, 16);
  buf.writeUInt16LE(0, 18);

  Buffer.from(".rsrc\0\0\0").copy(buf, COFF_HEADER);
  buf.writeUInt32LE(0, COFF_HEADER + 8);
  buf.writeUInt32LE(0, COFF_HEADER + 12);
  buf.writeUInt32LE(rsrcSize, COFF_HEADER + 16);
  buf.writeUInt32LE(sectionDataOff, COFF_HEADER + 20);
  buf.writeUInt32LE(relocTableOff, COFF_HEADER + 24);
  buf.writeUInt16LE(numRelocs, COFF_HEADER + 32);
  buf.writeUInt32LE(0x40000040, COFF_HEADER + 36);

  rsrcBuf.copy(buf, sectionDataOff);

  for (let i = 0; i < numRelocs; i++) {
    const off = relocTableOff + i * RELOC_SIZE;
    buf.writeUInt32LE(relocOffsets[i], off);
    buf.writeUInt32LE(0, off + 4);
    buf.writeUInt16LE(relocType, off + 8);
  }

  Buffer.from(".rsrc\0\0\0").copy(buf, symbolTableOff);
  buf.writeUInt32LE(0, symbolTableOff + 8);
  buf.writeInt16LE(1, symbolTableOff + 12);
  buf.writeUInt16LE(0, symbolTableOff + 14);
  buf.writeUInt8(3, symbolTableOff + 16);
  buf.writeUInt8(0, symbolTableOff + 17);

  buf.writeUInt32LE(4, stringTableOff);

  return buf;
}

function generateSyso(projectDir, pkg, goArch, sysoPath, iconPath) {
  const resources = [];

  if (iconPath) {
    const icoData = fs.readFileSync(iconPath);
    const images = parseIco(icoData);
    images.forEach((img, i) => resources.push({ type: RT_ICON, id: i + 1, data: img.data }));
    resources.push({ type: RT_GROUP_ICON, id: 1, data: buildGroupIconData(images) });
  }

  resources.push({ type: RT_VERSION, id: 1, data: buildVersionData(pkg) });

  fs.writeFileSync(sysoPath, buildCoff(resources, goArch));
}

module.exports = { resolveIconPath, generateSyso };
