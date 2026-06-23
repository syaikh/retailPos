<script>
  import { onMount } from 'svelte';
  import { getUsers, getRolesList, createUser, updateUser, deleteUser } from '$modules/admin';
  import { toast } from '$shared/stores/toast.svelte';
  import { debounce } from '$shared/utils/debounce';
  import { useAuthStore } from '$modules/auth';


  import { Button, Pagination } from '$shared/ui';
  import { Users } from 'lucide-svelte';
  import UserFormModal from './UserFormModal.svelte';
  import UserDeleteModal from './UserDeleteModal.svelte';
  import UserToolbar from './UserToolbar.svelte';
  import UserTable from './UserTable.svelte';

  const authStore = useAuthStore();

  let loading = $state(true);
  let users = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let roles = $state([]);
  let searchQuery = $state('');
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedUser = $state(null);
  let modalMode = $state('add');
  let saving = $state(false);
  let isSearching = $state(false);
  let isInitialMount = $state(true);

  let sortBy = $state('username');
  let sortDir = $state('asc');
  let filterRole = $state('all');
  let filterStatus = $state('all');

  let userRole = $derived(
    authStore.user?.role?.name ||
    (authStore.user?.role && typeof authStore.user?.role === 'object' ? authStore.user.role.name : authStore.user?.role) ||
    ''
  );
  let canCreate = $derived(['superadmin', 'admin'].includes(userRole));
  let canEdit = $derived(['superadmin', 'admin'].includes(userRole));
  let canDelete = $derived(userRole === 'superadmin');
  let canView = $derived(userRole !== 'cashier' && userRole !== '');
  let canEditSuperadmin = $derived(userRole === 'superadmin');
  let usernameHasInvalidChars = $derived(form.username.length > 0 && !/^[a-zA-Z0-9]+$/.test(form.username));

  let currentUserID = $derived(authStore.user?.id || 0);

  let prevSearchQuery = '';
  let prevOffset = 0;
  let prevLimit = 20;
  let prevSortBy = 'username';
  let prevSortDir = 'asc';
  let prevFilterRole = 'all';
  let prevFilterStatus = 'all';

  let form = $state({
    username: '',
    email: '',
    password: '',
    role_id: 0,
    is_active: true
  });



  async function fetchUsers(isSearch = false) {
    try {
      if (!isSearch) loading = true;
      const params = { limit, offset, search: searchQuery, sort: sortBy, dir: sortDir };
      if (filterRole !== 'all') params.role = filterRole;
      if (filterStatus !== 'all') params.status = filterStatus;
      const result = await getUsers(params);
      users = result.data;
      total = result.total;
    } catch {
      toast.error('Failed to load users');
    } finally {
      if (!isSearch) loading = false;
      isSearching = false;
    }
  }

  async function fetchRoles() {
    try {
      roles = await getRolesList();
      if (roles.length > 0 && form.role_id === 0) {
        form.role_id = roles[0].id;
      }
    } catch {
      toast.error('Failed to load roles');
    }
  }

  onMount(async () => {
    isInitialMount = true;
    await fetchRoles();
    await fetchUsers(false);
    isInitialMount = false;
  });

  const debouncedSearch = debounce(() => {
    offset = 0;
    prevOffset = 0;
    fetchUsers(true);
  }, 400);

  $effect(() => {
    const sq = searchQuery;
    if (isInitialMount) return;
    if (sq !== prevSearchQuery) {
      prevSearchQuery = sq;
      if (sq === '') {
        offset = 0;
        prevOffset = 0;
        isSearching = false;
        fetchUsers(false);
      } else {
        isSearching = true;
        debouncedSearch();
      }
    }
  });

  $effect(() => {
    const off = offset;
    const lim = limit;
    if (isInitialMount) return;
    if (off !== prevOffset || lim !== prevLimit) {
      prevOffset = off;
      prevLimit = lim;
      fetchUsers(false);
    }
  });

  $effect(() => {
    const sb = sortBy;
    const sd = sortDir;
    if (isInitialMount) return;
    if (sb !== prevSortBy || sd !== prevSortDir) {
      prevSortBy = sb;
      prevSortDir = sd;
      offset = 0;
      prevOffset = 0;
      fetchUsers(false);
    }
  });

  $effect(() => {
    const fr = filterRole;
    const fs = filterStatus;
    if (isInitialMount) return;
    if (fr !== prevFilterRole || fs !== prevFilterStatus) {
      prevFilterRole = fr;
      prevFilterStatus = fs;
      offset = 0;
      prevOffset = 0;
      fetchUsers(false);
    }
  });

  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
  }

  function toggleSort(key) {
    if (sortBy === key) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortBy = key;
      sortDir = 'asc';
    }
  }

  function clearAllFilters() {
    filterRole = 'all';
    filterStatus = 'all';
    sortBy = 'username';
    sortDir = 'asc';
  }

  function openAdd() {
    modalMode = 'add';
    form = { username: '', email: '', password: '', role_id: roles[0]?.id || 0, is_active: true };
    showModal = true;
  }

  function openEdit(user) {
    modalMode = 'edit';
    selectedUser = user;
    form = {
      username: user.username,
      email: user.email,
      password: '',
      role_id: user.role_id,
      is_active: user.is_active
    };
    showModal = true;
  }

  async function saveUser() {
    if (!form.username || !form.email || (modalMode === 'add' && !form.password)) {
      toast.error('Please fill all required fields');
      return;
    }

    try {
      saving = true;

      if (modalMode === 'add') {
        await createUser(form);
      } else {
        await updateUser(selectedUser.id, form);
      }

      toast.success(modalMode === 'add' ? 'User created' : 'User updated');
      showModal = false;
      await fetchUsers();
    } catch (error) {
      const message = error?.response?.data?.error || error?.message || 'Network error';
      toast.error(message);
    } finally {
      saving = false;
    }
  }

  async function confirmDelete() {
    if (!selectedUser) return;
    if (selectedUser.id === currentUserID) {
      toast.error('You cannot delete your own account');
      return;
    }
    try {
      await deleteUser(selectedUser.id);
      toast.success(`User "${selectedUser.username}" removed`);
      await fetchUsers();
    } catch {
      toast.error('Failed to delete user');
    } finally {
      showDeleteModal = false;
      selectedUser = null;
    }
  }
</script>

<div class="space-y-5">
  {#if !canView}
    <div class="card px-4 py-12 text-center">
      <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
        <Users size={32} class="text-text-muted" />
      </div>
      <p class="text-text-primary font-semibold mt-4">Access Denied</p>
      <p class="text-text-muted text-sm mt-1">You do not have permission to view users</p>
    </div>
  {:else}
    <UserToolbar
      bind:searchQuery
      {roles}
      bind:filterRole
      bind:filterStatus
      {canCreate}
      onadd={openAdd}
      onclearall={clearAllFilters}
    />

    <div class="card p-0 overflow-hidden">
      <UserTable
        {users}
        {loading}
        {searchQuery}
        {canEdit}
        {canDelete}
        {canEditSuperadmin}
        {currentUserID}
        bind:sortBy
        bind:sortDir
        onsort={toggleSort}
        onedit={openEdit}
        ondelete={(user) => { selectedUser = user; showDeleteModal = true; }}
      />

      {#if !loading && users.length > 0}
        <div class="p-4 bg-surface-subtle/30 border-t border-border/50">
          <Pagination
            {total}
            {limit}
            {offset}
            onPageChange={handlePageChange}
          />
        </div>
      {/if}
    </div>
  {/if}
</div>

<UserFormModal
  bind:open={showModal}
  {modalMode}
  bind:form
  {roles}
  bind:saving
  usernameHasInvalidChars={usernameHasInvalidChars}
  onsave={saveUser}
/>

<UserDeleteModal
  bind:open={showDeleteModal}
  username={selectedUser?.username ?? ''}
  oncancel={() => showDeleteModal = false}
  onconfirm={confirmDelete}
/>

