<!-- Copyright By https://github.com/Daymychen/art-design-pro/blob/main/src/components/core/views/login/LoginLeftView.vue -->
<script lang="ts" setup>
import { useThemeStore } from '@/store/modules/theme';

defineOptions({ name: 'WaveBg' });

const themeStore = useThemeStore();

function toggleThemeScheme() {
  if (themeStore.darkMode) {
    themeStore.setThemeScheme('light');
    return;
  }
  themeStore.setThemeScheme('dark');
}
</script>

<template>
  <div class="wave-bg">
    <!--
      装饰色体系：通过 CSS 自定义属性统一管理，亮色基于 --primary 色板，
      暗黑模式使用显式色值，避免色板翻转导致对比度不足。
    -->
    <div
      class="geometric-decorations"
      :style="{
        '--wv-fill': 'rgb(var(--primary-300-color) / 0.30)',
        '--wv-fill-strong': 'rgb(var(--primary-300-color) / 0.50)',
        '--wv-border': 'rgb(var(--primary-300-color) / 0.50)',
        '--wv-dot': 'rgb(var(--primary-300-color) / 0.45)',
        '--wv-sky': 'rgb(var(--primary-300-color) / 0.25)',
        '--wv-sun': '#fbbf24',
        '--wv-moon': '#e8d5b0',
        '--wv-fill-dark': 'rgb(var(--primary-600-color) / 0.25)',
        '--wv-fill-strong-dark': 'rgb(var(--primary-500-color) / 0.30)',
        '--wv-border-dark': 'rgb(var(--primary-500-color) / 0.30)',
        '--wv-dot-dark': 'rgb(var(--primary-500-color) / 0.35)',
        '--wv-sky-dark': '#1a1725',
        '--wv-square-blue': 'rgb(var(--primary-color) / 0.30)',
        '--wv-square-blue-dark': 'rgb(var(--primary-color) / 0.18)',
        '--wv-square-pink': 'rgb(var(--primary-color) / 0.15)',
        '--wv-square-pink-dark': 'rgb(var(--primary-color) / 0.10)',
        '--wv-square-purple': 'rgb(var(--primary-color) / 0.45)',
        '--wv-square-purple-dark': 'rgb(var(--primary-color) / 0.20)'
      }"
    >
      <!-- 基础几何形状 -->
      <div class="geo-element circle-outline animate-fade-in-up animate-delay-0s" />
      <div class="geo-element square-rotated animate-fade-in-left animate-delay-0s" />
      <div class="geo-element circle-small animate-fade-in-up animate-delay-0.3s" />

      <div class="geo-element square-bottom-right animate-fade-in-right animate-delay-0s" />

      <!-- 背景泡泡（太阳/月亮的"夜空"背板） -->
      <div class="geo-element bg-bubble animate-scale-in animate-delay-0.5s" />

      <!-- 太阳 / 月亮（点击切换亮暗主题） -->
      <div class="geo-element circle-top-right animate-fade-in-down animate-delay-0.5s" @click="toggleThemeScheme" />

      <!-- 装饰点 -->
      <div class="geo-element dot dot-top-left animate-bounce-in animate-delay-0s" />
      <div class="geo-element dot dot-top-right animate-bounce-in animate-delay-0s" />
      <div class="geo-element dot dot-center-right animate-bounce-in animate-delay-0s" />

      <!-- 叠加方块组 -->
      <div class="squares-group">
        <i class="geo-element square square-blue animate-fade-in-left-rotated-blue animate-delay-0.2s" />
        <i class="geo-element square square-pink animate-fade-in-left-rotated-pink animate-delay-0.4s" />
        <i class="geo-element square square-purple animate-fade-in-left-no-rotation animate-delay-0.6s" />
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
// ===================== 动画 + 定位（纯几何，不涉及颜色） =====================

.wave-bg {
  .geometric-decorations {
    .geo-element {
      position: absolute;
      opacity: 0;
      animation-fill-mode: forwards;
      animation-duration: 0.8s;
      animation-timing-function: cubic-bezier(0.25, 0.46, 0.45, 0.94);
    }

    // ---------- 动画 mixin ----------
    @mixin fadeAnimation($direction: '', $rotation: 0deg) {
      from {
        opacity: 0;

        @if $direction == 'up' {
          transform: translateY(30px) rotate($rotation);
        } @else if $direction == 'down' {
          transform: translateY(-30px) rotate($rotation);
        } @else if $direction == 'left' {
          transform: translateX(-30px) rotate($rotation);
        } @else if $direction == 'right' {
          transform: translateX(30px) rotate($rotation);
        }
      }

      to {
        opacity: 1;

        @if $direction == 'up' or $direction == 'down' {
          transform: translateY(0) rotate($rotation);
        } @else {
          transform: translateX(0) rotate($rotation);
        }
      }
    }

    // ---------- 动画定义 ----------
    @keyframes fadeInUp {
      @include fadeAnimation('up');
    }
    @keyframes fadeInDown {
      @include fadeAnimation('down');
    }
    @keyframes fadeInLeft {
      @include fadeAnimation('left');
    }
    @keyframes fadeInLeftRotated {
      @include fadeAnimation('left', -25deg);
    }
    @keyframes fadeInRight {
      @include fadeAnimation('right');
    }
    @keyframes fadeInRightRotated {
      @include fadeAnimation('right', 45deg);
    }
    @keyframes fadeInLeftRotatedBlue {
      @include fadeAnimation('left', -10deg);
    }
    @keyframes fadeInLeftRotatedPink {
      @include fadeAnimation('left', 10deg);
    }
    @keyframes fadeInLeftNoRotation {
      @include fadeAnimation('left');
    }

    @keyframes scaleIn {
      from {
        opacity: 0;
        transform: scale(0.8);
      }
      to {
        opacity: 1;
        transform: scale(1);
      }
    }

    @keyframes bounceIn {
      0% {
        opacity: 0;
        transform: scale(0.3);
      }
      50% {
        opacity: 1;
        transform: scale(1.05);
      }
      70% {
        transform: scale(0.9);
      }
      100% {
        opacity: 1;
        transform: scale(1);
      }
    }

    @keyframes lineGrow {
      from {
        opacity: 0;
      }
      to {
        opacity: 1;
      }
    }

    // ---------- 动画类 ----------
    .animate-fade-in-up {
      animation-name: fadeInUp;
    }
    .animate-fade-in-down {
      animation-name: fadeInDown;
    }
    .animate-fade-in-left {
      animation-name: fadeInLeft;
    }
    .animate-fade-in-right {
      animation-name: fadeInRight;
    }
    .animate-scale-in {
      animation-name: scaleIn;
      animation-duration: 1.2s;
    }
    .animate-bounce-in {
      animation-name: bounceIn;
      animation-duration: 0.6s;
    }
    .animate-fade-in-left-rotated-blue {
      animation-name: fadeInLeftRotatedBlue;
    }
    .animate-fade-in-left-rotated-pink {
      animation-name: fadeInLeftRotatedPink;
    }
    .animate-fade-in-left-no-rotation {
      animation-name: fadeInLeftNoRotation;
    }

    // ---------- 定位 & 形状（颜色来自 CSS 变量 / UnoCSS） ----------

    .circle-outline {
      top: 10%;
      left: 25%;
      width: 42px;
      height: 42px;
      border: 2px solid var(--wv-border);
      border-radius: 50%;
    }

    .square-rotated {
      top: 50%;
      left: 16%;
      width: 60px;
      height: 60px;
      background-color: var(--wv-fill);

      &.animate-fade-in-left {
        animation-name: fadeInLeftRotated;
      }
    }

    .circle-small {
      bottom: 26%;
      left: 30%;
      width: 18px;
      height: 18px;
      background-color: var(--wv-fill-strong);
      border-radius: 50%;
    }

    .square-bottom-right {
      right: 10%;
      bottom: 10%;
      width: 50px;
      height: 50px;
      background-color: var(--wv-fill-strong);

      &.animate-fade-in-right {
        animation-name: fadeInRightRotated;
      }
    }

    // ===================== 太阳 / 月亮 =====================
    .circle-top-right {
      top: 3%;
      right: 3%;
      z-index: 100;
      width: 50px;
      height: 50px;
      cursor: pointer;
      background-color: var(--wv-sun);
      border-radius: 50%;
      transition: all 0.3s;

      // hover 发光（亮色模式）
      &::after {
        position: absolute;
        top: 50%;
        left: 50%;
        width: 100%;
        height: 100%;
        content: '';
        background: linear-gradient(to right, #fcbb04, #fffc00);
        border-radius: 50%;
        opacity: 0;
        transition: all 0.5s;
        transform: translate(-50%, -50%);
      }

      &:hover {
        box-shadow: 0 0 36px #fffc00;

        &::after {
          opacity: 1;
        }
      }
    }

    // ---------- 暗黑模式：月亮 ----------
    .dark & .circle-top-right {
      background-color: var(--wv-moon);
      box-shadow: none;
      rotate: -48deg;

      // 月牙"裁剪"球：与夜空同色，覆盖月亮右侧形成弯月
      &::before {
        position: absolute;
        top: 0;
        left: 15px;
        width: 50px;
        height: 50px;
        content: '';
        background-color: var(--wv-sky-dark);
        border-radius: 50%;
        transition: all 0.3s ease-in-out;
      }

      &:hover {
        box-shadow: 0 0 18px rgba(255, 255, 255, 0.12) inset;

        &::before {
          left: 18px;
        }

        &::after {
          opacity: 0;
        }
      }
    }

    // ===================== 背景泡泡 =====================
    .bg-bubble {
      top: -120px;
      right: -120px;
      width: 360px;
      height: 360px;
      background-color: var(--wv-sky);
      border-radius: 50%;
    }

    // ---------- 装饰点 ----------
    .dot {
      width: 14px;
      height: 14px;
      background-color: var(--wv-dot);
      border-radius: 50%;

      &.dot-top-left {
        top: 140px;
        left: 100px;
      }

      &.dot-top-right {
        top: 140px;
        right: 120px;
      }

      &.dot-center-right {
        top: 46%;
        right: 22%;
        background-color: var(--wv-fill-strong);
      }
    }

    // ---------- 叠加方块组 ----------
    .squares-group {
      position: absolute;
      bottom: 18px;
      left: 20px;
      width: 140px;
      height: 140px;
      pointer-events: none;

      .square {
        position: absolute;
        display: block;
        border-radius: 8px;
        box-shadow: 0 8px 24px rgb(64 87 167 / 12%);

        &.square-blue {
          top: 12px;
          left: 30px;
          z-index: 2;
          width: 50px;
          height: 50px;
          background-color: var(--wv-square-blue);
        }

        &.square-pink {
          top: 30px;
          left: 48px;
          z-index: 1;
          width: 70px;
          height: 70px;
          background-color: var(--wv-square-pink);
        }

        &.square-purple {
          top: 66px;
          left: 86px;
          z-index: 3;
          width: 32px;
          height: 32px;
          background-color: var(--wv-square-purple);
        }
      }

      // 装饰线条
      &::after {
        position: absolute;
        top: 86px;
        left: 72px;
        width: 80px;
        height: 1px;
        content: '';
        background: linear-gradient(90deg, var(--el-color-primary-light-6), transparent);
        opacity: 0;
        transform: rotate(50deg);
        animation: lineGrow 0.8s cubic-bezier(0.25, 0.46, 0.45, 0.94) forwards;
        animation-delay: 1.2s;
      }
    }
  }

  // ===================== 响应式 =====================
  @media only screen and (max-width: 1600px) {
    width: 60vw;
  }

  @media only screen and (max-width: 1280px) {
    width: auto;
    height: auto;
    padding: 0;
    background: transparent;

    .geometric-decorations {
      display: none;
    }
  }
}

// ===================== 暗黑模式全局覆写 =====================
.dark .wave-bg {
  // 暗色底色（保持与原有观感一致）
  background-color: color-mix(in srgb, rgb(var(--primary-200-color)) 60%, #070707);

  @media only screen and (max-width: 1280px) {
    background: transparent;
  }

  .geometric-decorations {
    // 几何装饰 → 暗色色值
    .circle-outline {
      border-color: var(--wv-border-dark);
    }

    .square-rotated {
      background-color: var(--wv-fill-dark);
    }

    .circle-small {
      background-color: var(--wv-fill-strong-dark);
    }

    .square-bottom-right {
      background-color: var(--wv-fill-strong-dark);
    }

    .bg-bubble {
      background-color: var(--wv-sky-dark);
    }

    .dot {
      background-color: var(--wv-dot-dark);

      &.dot-center-right {
        background-color: var(--wv-fill-strong-dark);
      }

      &.dot-top-right {
        background-color: var(--wv-dot-dark);
      }
    }

    // 方块组暗色
    .squares-group {
      .square {
        box-shadow: none;

        &.square-blue {
          background-color: var(--wv-square-blue-dark);
        }

        &.square-pink {
          background-color: var(--wv-square-pink-dark);
        }

        &.square-purple {
          background-color: var(--wv-square-purple-dark);
        }
      }

      &::after {
        background: linear-gradient(90deg, var(--wv-fill-strong), transparent);
      }
    }
  }
}
</style>
