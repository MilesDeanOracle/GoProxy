<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

interface TrafficPoint {
  time: string
  uploadRate: number
  downloadRate: number
}

const props = defineProps<{
  data: TrafficPoint[]
}>()

const containerEl = ref<HTMLDivElement | null>(null)
const canvasEl = ref<HTMLCanvasElement | null>(null)
const tooltipEl = ref<HTMLDivElement | null>(null)

const PAD = { top: 42, right: 20, bottom: 26, left: 54 }
const COL_UP = '#3b82f6'
const COL_DN = '#f59e0b'

let w = 0
let h = 0
let hoverIdx = -1
let pulsePhase = 0
let animId: number | null = null
let resizeObserver: ResizeObserver | null = null
let cachedBW = 0
let cachedBH = 0
let dispX: number[] = []
let yAnimStart = -Infinity
let yTrackedTime = ''

function fmtBytes(v: number): string {
  if (v < 0.5) return '0 B'
  if (v < 1024) return `${Math.round(v)} B`
  const u = ['KB', 'MB', 'GB', 'TB']
  let n = v / 1024, i = 0
  for (; i < u.length - 1 && n >= 1024; i++) n /= 1024
  return `${n.toFixed(n >= 10 ? 1 : 2)} ${u[i]}`
}
function fmtRate(v: number) { return `${fmtBytes(v)}/s` }

function rgba(hex: string, a: number) {
  return `rgba(${parseInt(hex.slice(1, 3), 16)},${parseInt(hex.slice(3, 5), 16)},${parseInt(hex.slice(5, 7), 16)},${a})`
}

function niceNum(range: number, round: boolean) {
  if (range <= 0) return 1
  const e = Math.floor(Math.log10(range))
  const f = range / 10 ** e
  const n = round ? (f < 1.5 ? 1 : f < 3 ? 2 : f < 7 ? 5 : 10) : (f <= 1 ? 1 : f <= 2 ? 2 : f <= 5 ? 5 : 10)
  return n * 10 ** e
}

function yScale(vals: number[], ticks = 5) {
  const max = Math.max(0, ...vals)
  if (max === 0) return { min: 0, max: 1, step: 0.2 }
  const range = niceNum(max, false)
  const step = niceNum(range / (ticks - 1), true)
  return { min: 0, max: Math.ceil(max / step) * step, step }
}

// Catmull-Rom → Hermite subdivision: generates dense point array for silky curves
function subdivide(pts: { x: number; y: number }[], subs = 8): { x: number; y: number }[] {
  const n = pts.length
  if (n < 2) return [...pts]
  if (n === 2) {
    const out: { x: number; y: number }[] = []
    for (let s = 0; s <= subs; s++) {
      const t = s / subs
      out.push({ x: pts[0].x + (pts[1].x - pts[0].x) * t, y: pts[0].y + (pts[1].y - pts[0].y) * t })
    }
    return out
  }
  // Catmull-Rom tangents at each point
  const tg: { x: number; y: number }[] = []
  for (let i = 0; i < n; i++) {
    const p = pts[Math.max(0, i - 1)]
    const q = pts[Math.min(n - 1, i + 1)]
    tg.push({ x: (q.x - p.x) / 2, y: (q.y - p.y) / 2 })
  }
  const out: { x: number; y: number }[] = []
  for (let i = 0; i < n - 1; i++) {
    for (let s = 0; s < subs; s++) {
      const t = s / subs, t2 = t * t, t3 = t2 * t
      const h00 = 2 * t3 - 3 * t2 + 1
      const h10 = t3 - 2 * t2 + t
      const h01 = -2 * t3 + 3 * t2
      const h11 = t3 - t2
      out.push({
        x: h00 * pts[i].x + h10 * tg[i].x + h01 * pts[i + 1].x + h11 * tg[i + 1].x,
        y: h00 * pts[i].y + h10 * tg[i].y + h01 * pts[i + 1].y + h11 * tg[i + 1].y
      })
    }
  }
  out.push(pts[n - 1])
  return out
}

function tracePoly(ctx: CanvasRenderingContext2D, pts: { x: number; y: number }[]) {
  ctx.moveTo(pts[0].x, pts[0].y)
  for (let i = 1; i < pts.length; i++) ctx.lineTo(pts[i].x, pts[i].y)
}

function draw() {
  const canvas = canvasEl.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  const dpr = window.devicePixelRatio || 1
  w = canvas.clientWidth; h = canvas.clientHeight
  if (w <= 0 || h <= 0) return

  const bw = Math.round(w * dpr), bh = Math.round(h * dpr)
  if (bw !== cachedBW || bh !== cachedBH) {
    canvas.width = bw; canvas.height = bh
    cachedBW = bw; cachedBH = bh
  }
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, w, h)

  const data = props.data
  const pl = PAD.left, pr = w - PAD.right, pt = PAD.top, pb = h - PAD.bottom
  const pw = pr - pl, ph = pb - pt
  if (pw <= 0 || ph <= 0) return

  const allVals = [...data.map(d => d.uploadRate), ...data.map(d => d.downloadRate)]
  const ys = yScale(allVals)
  const yOf = (v: number) => pb - ((v - ys.min) / (ys.max - ys.min)) * ph

  // Grid
  ctx.textAlign = 'right'; ctx.textBaseline = 'middle'; ctx.font = '10px sans-serif'
  for (let v = ys.min; v <= ys.max + ys.step * 0.01; v += ys.step) {
    const y = yOf(v)
    ctx.strokeStyle = 'rgba(125,133,144,0.10)'; ctx.lineWidth = 1
    ctx.beginPath(); ctx.moveTo(pl, y); ctx.lineTo(pr, y); ctx.stroke()
    ctx.fillStyle = '#7d8590'; ctx.fillText(fmtBytes(v), pl - 6, y)
  }
  ctx.strokeStyle = '#2a3340'; ctx.lineWidth = 1
  ctx.beginPath(); ctx.moveTo(pl, pb); ctx.lineTo(pr, pb); ctx.stroke()

  if (data.length > 0) {
    ctx.textAlign = 'center'; ctx.textBaseline = 'top'
    const gap = Math.max(1, Math.floor(data.length / 6))
    for (let i = 0; i < data.length; i += gap) {
      const x = data.length > 1 ? pl + (i / (data.length - 1)) * pw : pl + pw / 2
      ctx.fillStyle = '#7d8590'; ctx.fillText(data[i].time, x, pb + 6)
    }
  }

  if (data.length < 2) { drawLegend(ctx); return }

  const n = data.length
  const tx = (i: number) => pl + (i / (n - 1)) * pw

  // Continuous x lerp — every frame moves 10% toward target
  while (dispX.length < n) dispX.push(tx(dispX.length))
  if (dispX.length > n) dispX.length = n
  for (let i = 0; i < n; i++) dispX[i] += (tx(i) - dispX[i]) * 0.1

  // Last point y: 500 ms smoothstep
  const lt = data[n - 1].time
  if (lt !== yTrackedTime) { yTrackedTime = lt; yAnimStart = performance.now() }
  const yp = Math.min(1, (performance.now() - yAnimStart) / 500)
  const ye = yp * yp * (3 - 2 * yp)
  const prevUpY = yOf(data[n - 2].uploadRate)
  const prevDnY = yOf(data[n - 2].downloadRate)
  const lastUpY = prevUpY + (yOf(data[n - 1].uploadRate) - prevUpY) * ye
  const lastDnY = prevDnY + (yOf(data[n - 1].downloadRate) - prevDnY) * ye
  const rawUp = Array.from({ length: n }, (_, i) =>
    i === n - 1 ? { x: dispX[i], y: lastUpY } : { x: dispX[i], y: yOf(data[i].uploadRate) }
  )
  const rawDn = Array.from({ length: n }, (_, i) =>
    i === n - 1 ? { x: dispX[i], y: lastDnY } : { x: dispX[i], y: yOf(data[i].downloadRate) }
  )
  const clamp = (p: { x: number; y: number }) => ({ x: p.x, y: Math.max(pt, Math.min(pb, p.y)) })
  const upSub = subdivide(rawUp).map(clamp)
  const dnSub = subdivide(rawDn).map(clamp)

  // Area fills
  drawArea(ctx, upSub, COL_UP, pb)
  drawArea(ctx, dnSub, COL_DN, pb)

  // Lines
  drawCurve(ctx, upSub, COL_UP)
  drawCurve(ctx, dnSub, COL_DN)

  // Pulse dots
  pulseDot(ctx, rawUp[rawUp.length - 1], COL_UP)
  pulseDot(ctx, rawDn[rawDn.length - 1], COL_DN)

  drawLegend(ctx)

  // Hover
  if (hoverIdx >= 0 && hoverIdx < n) {
    const hx = pl + (hoverIdx / (n - 1)) * pw
    ctx.save()
    ctx.strokeStyle = 'rgba(125,133,144,0.35)'; ctx.lineWidth = 1; ctx.setLineDash([4, 4])
    ctx.beginPath(); ctx.moveTo(hx, pt); ctx.lineTo(hx, pb); ctx.stroke()
    ctx.restore()
    hoverDot(ctx, hx, yOf(data[hoverIdx].uploadRate), COL_UP)
    hoverDot(ctx, hx, yOf(data[hoverIdx].downloadRate), COL_DN)
  }
}

function drawArea(ctx: CanvasRenderingContext2D, pts: { x: number; y: number }[], color: string, bottom: number) {
  const minY = Math.min(...pts.map(p => p.y))
  const grad = ctx.createLinearGradient(0, minY, 0, bottom)
  grad.addColorStop(0, rgba(color, 0.25))
  grad.addColorStop(0.6, rgba(color, 0.06))
  grad.addColorStop(1, rgba(color, 0))
  ctx.beginPath()
  ctx.moveTo(pts[0].x, bottom)
  ctx.lineTo(pts[0].x, pts[0].y)
  for (let i = 1; i < pts.length; i++) ctx.lineTo(pts[i].x, pts[i].y)
  ctx.lineTo(pts[pts.length - 1].x, bottom)
  ctx.closePath()
  ctx.fillStyle = grad
  ctx.fill()
}

function drawCurve(ctx: CanvasRenderingContext2D, pts: { x: number; y: number }[], color: string) {
  ctx.save()
  ctx.strokeStyle = color
  ctx.lineWidth = 1.8
  ctx.lineJoin = 'round'; ctx.lineCap = 'round'
  ctx.beginPath(); tracePoly(ctx, pts); ctx.stroke()
  ctx.restore()
}

function pulseDot(ctx: CanvasRenderingContext2D, pt: { x: number; y: number }, color: string) {
  const p = Math.sin(pulsePhase) * 0.5 + 0.5
  ctx.beginPath()
  ctx.arc(pt.x, pt.y, 3 + p * 8, 0, Math.PI * 2)
  ctx.strokeStyle = rgba(color, 0.3 * (1 - p))
  ctx.lineWidth = 1.5
  ctx.stroke()
  ctx.beginPath(); ctx.arc(pt.x, pt.y, 3, 0, Math.PI * 2)
  ctx.fillStyle = color; ctx.fill()
}

function hoverDot(ctx: CanvasRenderingContext2D, x: number, y: number, color: string) {
  ctx.beginPath(); ctx.arc(x, y, 7, 0, Math.PI * 2)
  ctx.fillStyle = rgba(color, 0.18); ctx.fill()
  ctx.beginPath(); ctx.arc(x, y, 4.5, 0, Math.PI * 2)
  ctx.fillStyle = color; ctx.fill()
  ctx.strokeStyle = '#161b22'; ctx.lineWidth = 2; ctx.stroke()
}

function drawLegend(ctx: CanvasRenderingContext2D) {
  const items = [{ text: '上传', color: COL_UP }, { text: '下载', color: COL_DN }]
  ctx.font = '12px sans-serif'; ctx.textBaseline = 'middle'
  const r = 4, g = 6, sp = 16
  let tw = 0
  for (const it of items) tw += r * 2 + g + ctx.measureText(it.text).width + sp
  tw -= sp
  let x = w - 10 - tw; const y = 12
  for (const it of items) {
    ctx.beginPath(); ctx.arc(x + r, y, r, 0, Math.PI * 2)
    ctx.fillStyle = it.color; ctx.fill()
    x += r * 2 + g
    ctx.fillStyle = '#7d8590'; ctx.textAlign = 'left'
    ctx.fillText(it.text, x, y)
    x += ctx.measureText(it.text).width + sp
  }
}

function onMouseMove(e: MouseEvent) {
  const canvas = canvasEl.value
  if (!canvas || props.data.length < 2) return
  const rect = canvas.getBoundingClientRect()
  const mx = e.clientX - rect.left
  const pw = w - PAD.left - PAD.right
  if (mx < PAD.left || mx > w - PAD.right) {
    if (hoverIdx !== -1) { hoverIdx = -1; hideTooltip() }
    return
  }
  hoverIdx = Math.max(0, Math.min(props.data.length - 1, Math.round(((mx - PAD.left) / pw) * (props.data.length - 1))))
  showTooltip(e)
}

function onMouseLeave() { hoverIdx = -1; hideTooltip() }

function showTooltip(e: MouseEvent) {
  const el = tooltipEl.value
  if (!el || hoverIdx < 0 || hoverIdx >= props.data.length) return
  const d = props.data[hoverIdx]
  el.innerHTML =
    `<div style="margin-bottom:4px;color:var(--muted, #7d8590)">${d.time}</div>` +
    `<div style="display:flex;align-items:center;gap:6px">` +
    `<span style="width:8px;height:8px;border-radius:50%;background:${COL_UP};display:inline-block"></span>` +
    `<span>上传: ${fmtRate(d.uploadRate)}</span></div>` +
    `<div style="display:flex;align-items:center;gap:6px;margin-top:2px">` +
    `<span style="width:8px;height:8px;border-radius:50%;background:${COL_DN};display:inline-block"></span>` +
    `<span>下载: ${fmtRate(d.downloadRate)}</span></div>`
  const container = containerEl.value
  if (container) {
    const cr = container.getBoundingClientRect()
    let tx = e.clientX - cr.left + 14
    if (tx + 170 > w) tx = e.clientX - cr.left - 170
    el.style.left = `${tx}px`; el.style.top = `${e.clientY - cr.top - 20}px`
  }
  el.style.display = 'block'
}

function hideTooltip() { if (tooltipEl.value) tooltipEl.value.style.display = 'none' }

function loop(time: number) {
  animId = requestAnimationFrame(loop)
  pulsePhase = (time % 2000) / 2000 * Math.PI * 2
  draw()
}

onMounted(() => {
  loop(performance.now())
  if (containerEl.value) {
    resizeObserver = new ResizeObserver(() => { cachedBW = 0 })
    resizeObserver.observe(containerEl.value)
  }
})

onBeforeUnmount(() => {
  if (animId !== null) cancelAnimationFrame(animId)
  resizeObserver?.disconnect()
})
</script>

<template>
  <div ref="containerEl" class="traffic-canvas-wrap">
    <canvas ref="canvasEl" @mousemove="onMouseMove" @mouseleave="onMouseLeave" />
    <div ref="tooltipEl" class="traffic-tooltip" />
  </div>
</template>

<style scoped>
.traffic-canvas-wrap {
  position: relative;
  width: 100%;
}

canvas {
  display: block;
  width: 100%;
  height: 100%;
  cursor: crosshair;
}

.traffic-tooltip {
  display: none;
  position: absolute;
  pointer-events: none;
  background: color-mix(in srgb, var(--bg3) 92%, transparent);
  backdrop-filter: blur(6px);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 8px 12px;
  font-size: 12px;
  color: var(--text);
  white-space: nowrap;
  z-index: 10;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
}
</style>
