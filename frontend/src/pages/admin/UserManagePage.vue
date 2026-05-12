<template>
  <div id="userManagePage">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-container">
        <div class="header-content">
          <h1 class="page-title">用户管理</h1>
          <p class="page-subtitle">管理系统中的所有用户</p>
        </div>
      </div>
    </div>

    <div class="container">
      <a-card :bordered="false" class="content-card">
        <!-- 搜索表单 -->
        <div class="search-section">
          <a-form layout="inline" :model="searchParams" @finish="doSearch" class="search-form">
            <a-form-item label="账号">
              <a-input v-model:value="searchParams.userAccount" placeholder="输入账号" class="search-input" />
            </a-form-item>
            <a-form-item label="用户名">
              <a-input v-model:value="searchParams.userName" placeholder="输入用户名" class="search-input" />
            </a-form-item>
            <a-form-item>
              <a-button type="primary" html-type="submit" class="search-btn">
                <template #icon>
                  <SearchOutlined />
                </template>
                搜索
              </a-button>
            </a-form-item>
          </a-form>
        </div>

        <a-divider />

        <!-- 表格 -->
        <a-table
          :columns="columns"
          :data-source="data"
          :pagination="pagination"
          @change="doTableChange"
          class="user-table"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'userAvatar'">
              <a-avatar :src="record.userAvatar" :size="48" class="user-avatar" />
            </template>
            <template v-else-if="column.dataIndex === 'userRole'">
              <a-tag v-if="record.userRole === 'admin'" color="purple" class="role-tag">
                管理员
              </a-tag>
              <a-tag v-else color="blue" class="role-tag">
                普通用户
              </a-tag>
            </template>
            <template v-else-if="column.dataIndex === 'createTime'">
              <span class="time-text">{{ dayjs(record.createTime).format('YYYY-MM-DD HH:mm:ss') }}</span>
            </template>
            <template v-else-if="column.key === 'action'">
              <a-popconfirm
                title="确定要删除此用户吗?"
                ok-text="确定"
                cancel-text="取消"
                @confirm="doDelete(record.id)"
              >
                <a-button type="link" danger class="delete-btn">删除</a-button>
              </a-popconfirm>
            </template>
          </template>
        </a-table>
      </a-card>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { deleteUser, listUserVoByPage } from '@/api/userController.ts'
import { message } from 'ant-design-vue'
import { SearchOutlined } from '@ant-design/icons-vue'
import dayjs from 'dayjs'

const columns = [
  {
    title: 'id',
    dataIndex: 'id',
  },
  {
    title: '账号',
    dataIndex: 'userAccount',
  },
  {
    title: '用户名',
    dataIndex: 'userName',
  },
  {
    title: '头像',
    dataIndex: 'userAvatar',
  },
  {
    title: '简介',
    dataIndex: 'userProfile',
  },
  {
    title: '用户角色',
    dataIndex: 'userRole',
  },
  {
    title: '创建时间',
    dataIndex: 'createTime',
  },
  {
    title: '操作',
    key: 'action',
  },
]

// 展示的数据
const data = ref<API.UserVO[]>([])
const total = ref(0)

// 搜索条件
const searchParams = reactive<API.UserQueryRequest>({
  pageNum: 1,
  pageSize: 10,
})

// 获取数据
const fetchData = async () => {
  const res = await listUserVoByPage({
    ...searchParams,
  })
  if (res.data.data) {
    data.value = res.data.data.records ?? []
    total.value = res.data.data.totalRow ?? 0
  } else {
    message.error('获取数据失败，' + res.data.message)
  }
}

// 分页参数
const pagination = computed(() => {
  return {
    current: searchParams.pageNum ?? 1,
    pageSize: searchParams.pageSize ?? 10,
    total: total.value,
    showSizeChanger: true,
    showTotal: (total: number) => `共 ${total} 条`,
  }
})

// 表格分页变化时的操作
const doTableChange = (page: { current: number; pageSize: number }) => {
  searchParams.pageNum = page.current
  searchParams.pageSize = page.pageSize
  fetchData()
}

// 搜索数据
const doSearch = () => {
  // 重置页码
  searchParams.pageNum = 1
  fetchData()
}

// 删除数据
const doDelete = async (id: string) => {
  if (!id) {
    return
  }
  const res = await deleteUser({ id: Number(id) })
  if (res.data.code === 0) {
    message.success('删除成功')
    // 刷新数据
    fetchData()
  } else {
    message.error('删除失败')
  }
}

// 页面加载时请求一次
onMounted(() => {
  fetchData()
})
</script>

<style scoped lang="scss">
#userManagePage {
  background: var(--color-background-secondary);
  min-height: 100vh;
  padding-bottom: 60px;

  .page-header {
    background: var(--gradient-hero);
    padding: 32px 20px;
    margin-bottom: 24px;
  }

  .header-container {
    max-width: 1200px;
    margin: 0 auto;
  }

  .header-content {
    color: var(--color-text);
  }

  .page-title {
    font-size: 28px;
    font-weight: 700;
    margin: 0 0 6px;
    letter-spacing: -0.5px;
    color: var(--color-text);
  }

  .page-subtitle {
    font-size: 14px;
    color: var(--color-text-secondary);
    margin: 0;
  }

  .container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 0 20px;
  }

  .content-card {
    border-radius: var(--radius-lg);
    border: 1px solid var(--color-border);
    box-shadow: none;
    background: white;

    :deep(.ant-card-body) {
      padding: 24px;
    }
  }

  .search-section {
    margin-bottom: 8px;
  }

  .search-form {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    align-items: flex-end;

    :deep(.ant-form-item) {
      margin-bottom: 0;
    }

    :deep(.ant-form-item-label > label) {
      font-weight: 500;
      font-size: 13px;
      color: var(--color-text-secondary);
    }
  }

  .search-input {
    width: 180px;
    border-radius: var(--radius-md);

    &:hover {
      border-color: var(--color-primary-light);
    }

    &:focus {
      border-color: var(--color-primary);
      box-shadow: 0 0 0 2px rgba(34, 197, 94, 0.1);
    }
  }

  .search-btn {
    border-radius: var(--radius-md);
    font-weight: 500;
    background: var(--gradient-primary) !important;
    border: none !important;
    color: white !important;
    box-shadow: var(--shadow-green) !important;
    transition: opacity var(--transition-normal) !important;

    &:hover,
    &:focus,
    &:active {
      background: var(--gradient-primary) !important;
      border: none !important;
      color: white !important;
      box-shadow: var(--shadow-green) !important;
      opacity: 0.92;
    }

    :deep(.ant-wave) {
      display: none;
    }
  }

  .user-table {
    :deep(.ant-table-thead > tr > th) {
      background: var(--color-background-secondary);
      font-weight: 600;
      font-size: 13px;
      color: var(--color-text-secondary);
      border-bottom: 1px solid var(--color-border);
      padding: 14px 16px;
    }

    :deep(.ant-table-tbody > tr > td) {
      padding: 16px;
      border-bottom: 1px solid var(--color-border-light);
    }

    :deep(.ant-table-tbody > tr:hover > td) {
      background: rgba(34, 197, 94, 0.02);
    }

    :deep(.ant-table-pagination) {
      margin: 16px 0 0;
    }
  }

  .user-avatar {
    border: 2px solid var(--color-border);
  }

  .role-tag {
    border-radius: var(--radius-full);
    font-weight: 500;
    font-size: 12px;
    padding: 2px 10px;
  }

  .time-text {
    color: var(--color-text-secondary);
    font-size: 13px;
  }

  .delete-btn {
    font-weight: 500;
    font-size: 13px;
    color: var(--color-error);
    padding: 4px 8px;

    &:hover {
      color: #DC2626;
    }
  }
}

@media (max-width: 768px) {
  #userManagePage {
    .page-header {
      padding: 24px 20px;
    }

    .page-title {
      font-size: 22px;
    }

    .search-form {
      flex-direction: column;
      align-items: stretch;

      :deep(.ant-form-item) {
        width: 100%;
      }
    }

    .search-input {
      width: 100%;
    }
  }
}
</style>
