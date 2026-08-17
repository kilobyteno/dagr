import AppKit
import Foundation

guard CommandLine.arguments.count == 3 else {
  fputs("usage: pad-app-icon.swift <src.png> <dst.png>\n", stderr)
  exit(2)
}

let srcPath = CommandLine.arguments[1]
let dstPath = CommandLine.arguments[2]
let canvasSize = 1024
// Leave margin so the Dock squircle matches other macOS apps optically.
let contentScale = 0.8

guard
  let source = NSImage(contentsOfFile: srcPath),
  let cgSource = source.cgImage(forProposedRect: nil, context: nil, hints: nil)
else {
  fputs("Could not read source icon\n", stderr)
  exit(1)
}

let content = Int((Double(canvasSize) * contentScale).rounded())
let origin = (canvasSize - content) / 2
let colorSpace = CGColorSpaceCreateDeviceRGB()
let bytesPerRow = canvasSize * 4
var data = [UInt8](repeating: 0, count: bytesPerRow * canvasSize)

guard
  let ctx = CGContext(
    data: &data,
    width: canvasSize,
    height: canvasSize,
    bitsPerComponent: 8,
    bytesPerRow: bytesPerRow,
    space: colorSpace,
    bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue
  )
else {
  fputs("Could not create bitmap context\n", stderr)
  exit(1)
}

// Opaque white plate — transparency makes Dock icons look ghosted.
ctx.setFillColor(CGColor(red: 1, green: 1, blue: 1, alpha: 1))
ctx.fill(CGRect(x: 0, y: 0, width: canvasSize, height: canvasSize))
ctx.interpolationQuality = .high
ctx.draw(
  cgSource,
  in: CGRect(x: origin, y: origin, width: content, height: content)
)

guard
  let outImage = ctx.makeImage()
else {
  fputs("Could not finalise bitmap\n", stderr)
  exit(1)
}

let rep = NSBitmapImageRep(cgImage: outImage)
guard let png = rep.representation(using: .png, properties: [:]) else {
  fputs("Could not encode PNG\n", stderr)
  exit(1)
}

do {
  try png.write(to: URL(fileURLWithPath: dstPath))
} catch {
  fputs("Could not write \(dstPath): \(error)\n", stderr)
  exit(1)
}
