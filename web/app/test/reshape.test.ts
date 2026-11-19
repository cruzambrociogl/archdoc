// Reshaping a boundary: what it holds is laid out again to the shape pulled, and only box
// positions come out — the boundary still follows its boxes.

import { test } from 'node:test'
import assert from 'node:assert/strict'
import { arrangeDraft, reshape, reshapeStart } from '../src/screens/explorer/scene.ts'
import type { Layout } from '../src/api.ts'

const box = (id: string, x: number, y: number) => ({ id, rect: { x, y, w: 120, h: 40 } })

// Six boxes in one tall column inside "app", the way a sparse component view comes out.
const tall: Layout = {
  version: 1,
  width: 200,
  height: 700,
  boxes: ['a', 'b', 'c', 'd', 'e', 'f'].map((id, i) => box(id, 20, 40 + i * 100)),
  groups: [{ name: 'app', rect: { x: 10, y: 10, w: 140, h: 580 }, label_at: { x: 20, y: 20 } }],
  paths: [],
}

test('pulling the side out lays the boxes out in rows that fit the width', () => {
  const s = reshapeStart(tall, tall, 'app')!
  assert.deepEqual(s.pad, { left: 10, top: 30, right: 10, bottom: 10 })
  // Room for three boxes: 3×120 + 2×44 = 448, plus the pads.
  const d = reshape(s, tall, tall, { x: 10, y: 10, w: 470, h: 580 }, 'width')
  assert.deepEqual(d.a, { x: 20, y: 40 })
  assert.deepEqual(d.b, { x: 184, y: 40 })
  assert.deepEqual(d.c, { x: 348, y: 40 })
  assert.deepEqual(d.d, { x: 20, y: 144 })
  assert.deepEqual(d.f, { x: 348, y: 144 })

  // The boundary follows: as wide as pulled, as tall as two rows.
  const g = arrangeDraft(tall, d).groups![0].rect
  assert.deepEqual(g, { x: 10, y: 10, w: 468, h: 184 })
})

test('pulled narrower than a box, each box gets a row of its own', () => {
  const s = reshapeStart(tall, tall, 'app')!
  const d = reshape(s, tall, tall, { x: 10, y: 10, w: 50, h: 580 }, 'width')
  assert.equal(new Set(Object.values(d).map((p) => p.x)).size, 1)
})

test('pulling the bottom up lays them out in columns that fit the height', () => {
  const s = reshapeStart(tall, tall, 'app')!
  // Room for two boxes a column: 2×40 + 64 = 144.
  const d = reshape(s, tall, tall, { x: 10, y: 10, w: 140, h: 190 }, 'height')
  assert.deepEqual(d.a, { x: 20, y: 40 })
  assert.deepEqual(d.b, { x: 20, y: 144 })
  assert.deepEqual(d.c, { x: 184, y: 40 })
  assert.deepEqual(d.f, { x: 348, y: 144 })
})

test('a boundary inside moves as one piece, with its boxes', () => {
  const nested: Layout = {
    ...tall,
    boxes: [box('a', 30, 50), box('b', 30, 150), box('c', 20, 300)],
    groups: [
      { name: 'app', rect: { x: 10, y: 10, w: 160, h: 340 }, label_at: { x: 20, y: 20 } },
      { name: 'net', rect: { x: 20, y: 30, w: 140, h: 170 }, label_at: { x: 30, y: 40 } },
    ],
  }
  const s = reshapeStart(nested, nested, 'app')!
  assert.equal(s.items.length, 2)
  const d = reshape(s, nested, nested, { x: 10, y: 10, w: 600, h: 340 }, 'width')
  // "net" stays first and where it was; its boxes keep their places inside it.
  assert.deepEqual(d.a, { x: 30, y: 50 })
  assert.deepEqual(d.b, { x: 30, y: 150 })
  // "c" moves up beside it.
  assert.deepEqual(d.c, { x: 20 + 140 + 44, y: 30 })
})

test('a boundary that holds nothing cannot be reshaped', () => {
  const empty: Layout = { ...tall, boxes: [] }
  assert.equal(reshapeStart(empty, empty, 'app'), undefined)
})
