<script setup lang="ts">
import { ref } from 'vue';
import {useRouter} from "vue-router";

const router = useRouter();

const room = ref('');
const name = ref('');
const peer = ref('');

const onSubmit = () => {
  router.push({
    path: '/',
    query: {
      peerId: peer.value,
      userId: name.value,
      meetingId: room.value,
    },
  })
};
</script>

<template>
  <div class="p-grid" style="padding-top: 80px">
    <form class="p-col-6 p-offset-3" @submit.prevent="onSubmit">
      <Card>
        <template #title>
          Присоединиться к комнате
        </template>
        <template #content>
          <div class="p-fluid">
            <div class="p-field">
              <label for="name">Ваше имя</label>
              <InputText id="name" type="text" v-model="name" />
            </div>
            <div class="p-field">
              <label for="peer">Собеседник</label>
              <InputText id="peer" type="text" v-model="peer" />
            </div>
            <div class="p-field">
              <label for="room">Комната</label>
              <InputText id="room" type="text" v-model="room" />
            </div>
          </div>
        </template>
        <template #footer>
          <Button label="Присоединиться" :disabled="!name || !room || !peer ? 'disabled' : undefined" type="submit" />
        </template>
      </Card>
    </form>
  </div>
</template>
