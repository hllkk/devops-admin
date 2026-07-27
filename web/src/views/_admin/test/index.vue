<script setup lang="ts">
import { ref } from 'vue';

const count = ref(0);
</script>

<template>
  <div>
    <p>当前计数: {{ count }}</p>
    <button @click="count++">增加</button>
  </div>
</template>
