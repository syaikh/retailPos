<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { useAuthStore } from '$modules/auth';
  import { getPricingRules, createPricingRule, updatePricingRule, deletePricingRule, submitPricingRule, approvePricingRule, rejectPricingRule, searchProducts, getCustomerGroups, getStores, checkConflicts } from '../services/pricing-service';
  import { getCategories, getBrands, getProductsByIds } from '$modules/product/services/product-service';
  import type { PricingRule } from '../types';
  import type { ConflictRule } from '../services/pricing-service';
  import { Button, Input, Modal, Pagination, ConfirmDeleteModal, Badge, ImportWizard } from '$shared/ui';
  import { Loader2, AlertTriangle } from 'lucide-svelte';
  import { labels, t } from '$shared/i18n';
  import { useSortable } from '$shared/composables/useSortable.svelte';
  import PricingRulesToolbar from './PricingRulesToolbar.svelte';
  import PricingRulesTable from './PricingRulesTable.svelte';
  import PricingRuleDetailDrawer from './PricingRuleDetailDrawer.svelte';
  import PriceSimulationModal from './PriceSimulationModal.svelte';

  const authStore = useAuthStore();

  let loading = $state(true);
  let rules = $state<PricingRule[]>([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let showImportWizard = $state(false);
  let showSimulation = $state(false);
  let selectedRule = $state<PricingRule | null>(null);
  let modalMode = $state<'add' | 'edit'>('add');
  let saving = $state(false);
  const { sortState, handleSort } = useSortable('name', 'asc', fetchRules);
  let statusFilter = $state('all');
  let approvalFilter = $state('all');
  let typeFilter = $state('all');
  let methodFilter = $state('all');
  let showDetailDrawer = $state(false);
  let detailDrawerRule = $state<PricingRule | null>(null);

  let customerGroups = $state<{ id: number; name: string }[]>([]);
  let stores = $state<{ id: number; name: string }[]>([]);
  let productSearchResults = $state<{ id: number; name: string; sku: string; price: number }[]>([]);
  let productSearchQuery = $state('');
  let productSearchTimeout: ReturnType<typeof setTimeout> | null = null;
  let selectedProductName = $state('');
  let categories = $state<{ id: number; name: string }[]>([]);
  let brands = $state<{ id: number; name: string }[]>([]);
  let productNames = $state<Map<number, string>>(new Map());

  let targetNames = $derived((() => {
    const map = new Map<string, string>();
    for (const c of categories) map.set(`category:${c.id}`, c.name);
    for (const b of brands) map.set(`brand:${b.id}`, b.name);
    for (const [id, name] of productNames) map.set(`product:${id}`, name);
    return map;
  })());
  let categorySearchQuery = $state('');
  let brandSearchQuery = $state('');
  let categorySearchResults = $state<{ id: number; name: string }[]>([]);
  let brandSearchResults = $state<{ id: number; name: string }[]>([]);
  let categorySearchTimeout: ReturnType<typeof setTimeout> | null = null;
  let brandSearchTimeout: ReturnType<typeof setTimeout> | null = null;
  let selectedCategoryName = $state('');
  let selectedBrandName = $state('');
  let formErrors = $state<Record<string, string>>({});
  let showErrors = $state(false);
  let conflictRules = $state<ConflictRule[]>([]);
  let checkingConflicts = $state(false);
  let showConflictWarning = $state(false);

  const dayOptions = [
    { value: 'mon', label: labels.dayMon },
    { value: 'tue', label: labels.dayTue },
    { value: 'wed', label: labels.dayWed },
    { value: 'thu', label: labels.dayThu },
    { value: 'fri', label: labels.dayFri },
    { value: 'sat', label: labels.daySat },
    { value: 'sun', label: labels.daySun }
  ];

  let form = $state({
    product_id: null as number | null,
    category_id: null as number | null,
    brand_id: null as number | null,
    pricing_type: 'default' as string,
    pricing_method: 'fixed_price' as string,
    pricing_value: 0,
    name: '',
    minimum_quantity: 1,
    maximum_quantity: '' as number | string,
    priority: 0,
    customer_group_id: null as number | null,
    store_id: null as number | null,
    recurrence_days: [] as string[],
    time_from: '',
    time_to: '',
    allow_combine: false,
    is_active: true,
    effective_from: '',
    effective_until: ''
  });

  let canCreate = $derived((authStore.user?.permissions || []).includes('pricing.create'));
  let canEdit = $derived((authStore.user?.permissions || []).includes('pricing.update'));
  let canDelete = $derived((authStore.user?.permissions || []).includes('pricing.delete'));

  const pricingTypes = [
    { value: 'default', label: labels.tipeDefault, description: labels.tipeDefaultDesc },
    { value: 'special_price', label: labels.hargaKhusus, description: labels.hargaKhususDesc },
    { value: 'promotion', label: labels.promosi, description: labels.promosiDesc }
  ];

  const pricingMethods = [
    { value: 'fixed_price', label: labels.hargaTetap, description: labels.fixedPriceHelper },
    { value: 'discount_percent', label: labels.methodDiscountPercent, description: labels.discountPercentHelper },
    { value: 'discount_amount', label: labels.methodDiscountAmount, description: labels.discountAmountHelper },
    { value: 'markup_percent', label: labels.methodMarkupPercent, description: labels.markupPercentHelper }
  ];

  let typeLabel = $derived(typeFilter === 'all' ? labels.semuaTipe : pricingTypes.find(t => t.value === typeFilter)?.label || typeFilter);
  let methodLabel = $derived(methodFilter === 'all' ? labels.semuaMetode : pricingMethods.find(m => m.value === methodFilter)?.label || methodFilter);

  let sortedRules = $derived.by(() => {
    const sorted = [...rules];
    sorted.sort((a, b) => {
      let av: any = a[sortState.sortBy as keyof PricingRule];
      let bv: any = b[sortState.sortBy as keyof PricingRule];
      if (typeof av === 'string') av = av.toLowerCase();
      if (typeof bv === 'string') bv = bv.toLowerCase();
      if (av < bv) return sortState.sortDir === 'asc' ? -1 : 1;
      if (av > bv) return sortState.sortDir === 'asc' ? 1 : -1;
      return 0;
    });
    return sorted;
  });

  async function fetchRules() {
    loading = true;
    const params: any = { limit, offset, search: searchQuery, sort_by: sortState.sortBy, sort_dir: sortState.sortDir };
    if (statusFilter === 'active') params.is_active = true;
    else if (statusFilter === 'inactive') params.is_active = false;
    if (approvalFilter !== 'all') params.status = approvalFilter;
    if (typeFilter !== 'all') params.pricing_type = typeFilter;
    if (methodFilter !== 'all') params.pricing_method = methodFilter;
    const result = await getPricingRules(params);
    rules = result.data;
    total = result.total;
    loading = false;

    const uncachedIds = rules.filter(r => r.product_id).map(r => r.product_id!).filter(id => !productNames.has(id));
    if (uncachedIds.length > 0) {
      const [products, cg, st, cat, br] = await Promise.all([
        getProductsByIds(uncachedIds),
        customerGroups.length === 0 ? getCustomerGroups() : Promise.resolve(customerGroups),
        stores.length === 0 ? getStores() : Promise.resolve(stores),
        categories.length === 0 ? getCategories() : Promise.resolve(categories),
        brands.length === 0 ? getBrands() : Promise.resolve(brands),
      ]);
      const next = new Map(productNames);
      for (const [id, product] of products) {
        next.set(id, product.name);
      }
      productNames = next;
      customerGroups = cg;
      stores = st;
      categories = cat;
      brands = br;
    }
  }

  function resetForm() {
    form = {
      product_id: null, category_id: null, brand_id: null,
      pricing_type: 'default', pricing_method: 'fixed_price', pricing_value: 0,
      name: '', minimum_quantity: 1, maximum_quantity: '', priority: 0,
      customer_group_id: null, store_id: null, recurrence_days: [],
      time_from: '', time_to: '', allow_combine: false, is_active: true,
      effective_from: '', effective_until: ''
    };
    selectedProductName = '';
    productSearchResults = [];
    productSearchQuery = '';
    selectedCategoryName = '';
    categorySearchResults = [];
    categorySearchQuery = '';
    selectedBrandName = '';
    brandSearchResults = [];
    brandSearchQuery = '';
  }

  function openAdd() {
    modalMode = 'add';
    resetForm();
    formErrors = {};
    showErrors = false;
    showModal = true;
  }

  function openEdit(rule: PricingRule) {
    modalMode = 'edit';
    selectedRule = rule;
    form = {
      product_id: rule.product_id || null,
      category_id: rule.category_id || null,
      brand_id: rule.brand_id || null,
      pricing_type: rule.pricing_type,
      pricing_method: rule.pricing_method,
      pricing_value: rule.pricing_value,
      name: rule.name,
      minimum_quantity: rule.minimum_quantity,
      maximum_quantity: rule.maximum_quantity || '',
      priority: rule.priority,
      customer_group_id: rule.customer_group_id || null,
      store_id: rule.store_id || null,
      recurrence_days: rule.recurrence_days || [],
      time_from: rule.time_from || '',
      time_to: rule.time_to || '',
      allow_combine: rule.allow_combine || false,
      is_active: rule.is_active,
      effective_from: rule.effective_from ? String(rule.effective_from).split('T')[0] : '',
      effective_until: rule.effective_until ? String(rule.effective_until).split('T')[0] : ''
    };
    selectedProductName = rule.product_id ? t('productNumber', { value: rule.product_id }) : '';
    selectedCategoryName = rule.category_id ? categories.find(c => c.id === rule.category_id)?.name || t('categoryNumber', { value: rule.category_id }) : '';
    selectedBrandName = rule.brand_id ? brands.find(b => b.id === rule.brand_id)?.name || t('brandNumber', { value: rule.brand_id }) : '';
    formErrors = {};
    showErrors = false;
    showModal = true;
  }

  function openDelete(rule: PricingRule) {
    selectedRule = rule;
    showDeleteModal = true;
  }

  function handleDuplicate(rule: PricingRule) {
    modalMode = 'add';
    selectedRule = null;
    form = {
      product_id: rule.product_id || null,
      category_id: rule.category_id || null,
      brand_id: rule.brand_id || null,
      pricing_type: rule.pricing_type,
      pricing_method: rule.pricing_method,
      pricing_value: rule.pricing_value,
      name: `${rule.name}${labels.salinanSuffix}`,
      minimum_quantity: rule.minimum_quantity,
      maximum_quantity: rule.maximum_quantity || '',
      priority: rule.priority,
      customer_group_id: rule.customer_group_id || null,
      store_id: rule.store_id || null,
      recurrence_days: rule.recurrence_days ? [...rule.recurrence_days] : [],
      time_from: rule.time_from || '',
      time_to: rule.time_to || '',
      allow_combine: rule.allow_combine,
      is_active: rule.is_active,
      effective_from: '',
      effective_until: ''
    };
    selectedProductName = rule.product_id ? t('productNumber', { value: rule.product_id }) : '';
    selectedCategoryName = rule.category_id ? categories.find(c => c.id === rule.category_id)?.name || t('categoryNumber', { value: rule.category_id }) : '';
    selectedBrandName = rule.brand_id ? brands.find(b => b.id === rule.brand_id)?.name || t('brandNumber', { value: rule.brand_id }) : '';
    formErrors = {};
    showErrors = false;
    showModal = true;
  }

  function handleProductSearch() {
    if (productSearchTimeout) clearTimeout(productSearchTimeout);
    productSearchTimeout = setTimeout(async () => {
      if (productSearchQuery.length < 2) { productSearchResults = []; return; }
      productSearchResults = await searchProducts(productSearchQuery, 10);
    }, 300);
  }

  function selectProduct(product: { id: number; name: string; sku: string }) {
    form.product_id = product.id;
    selectedProductName = `${product.name} (${product.sku})`;
    productSearchResults = [];
    productSearchQuery = '';
  }

  function clearProduct() {
    form.product_id = null;
    selectedProductName = '';
  }

  function handleCategorySearch() {
    if (categorySearchTimeout) clearTimeout(categorySearchTimeout);
    categorySearchTimeout = setTimeout(() => {
      const q = categorySearchQuery.toLowerCase();
      if (q.length < 1) { categorySearchResults = []; return; }
      categorySearchResults = categories.filter(c => c.name.toLowerCase().includes(q)).slice(0, 10);
    }, 200);
  }

  function selectCategory(cat: { id: number; name: string }) {
    form.category_id = cat.id;
    selectedCategoryName = cat.name;
    categorySearchResults = [];
    categorySearchQuery = '';
  }

  function clearCategory() {
    form.category_id = null;
    selectedCategoryName = '';
  }

  function handleBrandSearch() {
    if (brandSearchTimeout) clearTimeout(brandSearchTimeout);
    brandSearchTimeout = setTimeout(() => {
      const q = brandSearchQuery.toLowerCase();
      if (q.length < 1) { brandSearchResults = []; return; }
      brandSearchResults = brands.filter(b => b.name.toLowerCase().includes(q)).slice(0, 10);
    }, 200);
  }

  function selectBrand(brand: { id: number; name: string }) {
    form.brand_id = brand.id;
    selectedBrandName = brand.name;
    brandSearchResults = [];
    brandSearchQuery = '';
  }

  function clearBrand() {
    form.brand_id = null;
    selectedBrandName = '';
  }

  function toggleDay(day: string) {
    if (form.recurrence_days.includes(day)) {
      form.recurrence_days = form.recurrence_days.filter(d => d !== day);
    } else {
      form.recurrence_days = [...form.recurrence_days, day];
    }
  }

  async function runConflictCheck(): Promise<ConflictRule[]> {
    checkingConflicts = true;
    try {
      const resp = await checkConflicts({
        product_id: form.product_id,
        category_id: form.category_id,
        brand_id: form.brand_id,
        pricing_type: form.pricing_type,
        pricing_method: form.pricing_method,
        pricing_value: form.pricing_value,
        minimum_quantity: form.minimum_quantity || 1,
        maximum_quantity: form.maximum_quantity ? Number(form.maximum_quantity) : undefined,
        priority: form.priority,
        exclude_id: modalMode === 'edit' ? selectedRule?.id : undefined,
      });
      return resp.data || [];
    } catch {
      return [];
    } finally {
      checkingConflicts = false;
    }
  }

  function buildPayload(): any {
    const payload: any = { ...form };
    if (payload.effective_from) {
      payload.effective_from = payload.effective_from + 'T00:00:00+07:00';
    } else {
      delete payload.effective_from;
    }
    if (payload.effective_until) {
      payload.effective_until = payload.effective_until + 'T23:59:59+07:00';
    } else {
      delete payload.effective_until;
    }
    if (payload.maximum_quantity === undefined || payload.maximum_quantity === '') delete payload.maximum_quantity;
    if (payload.customer_group_id === null) delete payload.customer_group_id;
    if (payload.store_id === null) delete payload.store_id;
    if (!payload.time_from) delete payload.time_from;
    if (!payload.time_to) delete payload.time_to;
    if (payload.recurrence_days.length === 0) delete payload.recurrence_days;
    return payload;
  }

  async function doSave() {
    saving = true;
    const payload = buildPayload();

    let result: { ok: boolean; error?: string };
    if (modalMode === 'add') {
      result = await createPricingRule(payload);
    } else {
      result = await updatePricingRule(selectedRule!.id, payload);
    }
    saving = false;

    if (result.ok) {
      toast.success(modalMode === 'add' ? labels.ruleCreated : labels.ruleUpdated);
      showModal = false;
      showConflictWarning = false;
      conflictRules = [];
      fetchRules();
    } else {
      toast.error(result.error || labels.failedToSaveRule);
    }
  }

  async function saveRule(e: Event) {
    e.preventDefault();
    const errors = validateForm();
    formErrors = errors;
    showErrors = true;
    if (Object.keys(errors).length > 0) {
      toast.error(Object.values(errors)[0]);
      return;
    }

    if (!showConflictWarning) {
      const conflicts = await runConflictCheck();
      if (conflicts.length > 0) {
        conflictRules = conflicts;
        showConflictWarning = true;
        return;
      }
    }

    showConflictWarning = false;
    conflictRules = [];
    doSave();
  }

  function handleRowClick(rule: PricingRule) {
    detailDrawerRule = rule;
    showDetailDrawer = true;
  }

  function handleDetailDrawerClose() {
    showDetailDrawer = false;
  }

  async function confirmDelete() {
    if (!selectedRule) return;
    const ok = await deletePricingRule(selectedRule.id);
    if (ok) {
      toast.success(labels.ruleDeleted);
      showDeleteModal = false;
      fetchRules();
    } else {
      toast.error(labels.failedToDeleteRule);
    }
  }

  function handlePageChange(newOffset: number, newLimit: number) {
    offset = newOffset;
    limit = newLimit;
    fetchRules();
  }

  function formatPrice(v: number): string {
    return v?.toLocaleString('id-ID') || '0';
  }

  function getMethodConfig(method: string) {
    switch (method) {
      case 'fixed_price': return { prefix: 'Rp', suffix: '', placeholder: labels.fixedPricePlaceholder, helper: labels.fixedPriceHelper };
      case 'discount_percent': return { prefix: '', suffix: '%', placeholder: labels.discountPercentPlaceholder, helper: labels.discountPercentHelper };
      case 'discount_amount': return { prefix: 'Rp', suffix: '', placeholder: labels.discountAmountPlaceholder, helper: labels.discountAmountHelper };
      case 'markup_percent': return { prefix: '', suffix: '%', placeholder: labels.markupPercentPlaceholder, helper: labels.markupPercentHelper };
      default: return { prefix: '', suffix: '', placeholder: '0', helper: '' };
    }
  }

  function selectAllDays() {
    form.recurrence_days = dayOptions.map(d => d.value);
  }
  function selectWorkDays() {
    form.recurrence_days = ['mon', 'tue', 'wed', 'thu', 'fri'];
  }
  function selectWeekend() {
    form.recurrence_days = ['sat', 'sun'];
  }
  function clearDays() {
    form.recurrence_days = [];
  }

  const allDaysSelected = $derived(form.recurrence_days.length === dayOptions.length);
  const workDaysSelected = $derived(['mon', 'tue', 'wed', 'thu', 'fri'].every(d => form.recurrence_days.includes(d)));
  const weekendSelected = $derived(['sat', 'sun'].every(d => form.recurrence_days.includes(d)));

  function formatDayRange(days: string[]): string {
    if (!days || days.length === 0) return labels.setiapHari;
    if (days.length === 7) return labels.setiapHari;
    const shortMap: Record<string, string> = { mon: labels.dayMonShort, tue: labels.dayTueShort, wed: labels.dayWedShort, thu: labels.dayThuShort, fri: labels.dayFriShort, sat: labels.daySatShort, sun: labels.daySunShort };
    const ordered = dayOptions.map(d => d.value).filter(d => days.includes(d));
    if (ordered.length === 0) return '-';
    const names = ordered.map(d => shortMap[d] || d);
    if (names.length <= 3) return names.join(', ');
    return t('dayRangeCount', { value: `${names[0]} - ${names[names.length - 1]} ${names.length}` });
  }

  let summaryPreview = $derived.by(() => {
    const method = getMethodConfig(form.pricing_method);
    let valueStr = '';
    if (form.pricing_method === 'fixed_price') valueStr = `Rp${formatPrice(form.pricing_value)}`;
    else if (form.pricing_method === 'discount_percent') valueStr = `${form.pricing_value}%`;
    else if (form.pricing_method === 'discount_amount') valueStr = `-Rp${formatPrice(form.pricing_value)}`;
    else if (form.pricing_method === 'markup_percent') valueStr = `+${form.pricing_value}%`;

    const catLabel = selectedCategoryName || '-';
    const brandLabel = selectedBrandName || '-';
    const cgLabel = form.customer_group_id ? customerGroups.find(cg => cg.id === form.customer_group_id)?.name || labels.dipilih : labels.semuaGroup;
    const storeLabel = form.store_id ? stores.find(s => s.id === form.store_id)?.name || labels.dipilih : labels.semuaOutlet;
    const qtyMax = form.maximum_quantity ? form.maximum_quantity : labels.tanpaBatas;
    const days = formatDayRange(form.recurrence_days);
    const timeStr = form.time_from && form.time_to ? `${form.time_from} - ${form.time_to}` : form.time_from ? `${form.time_from} - ...` : labels.sepanjangHari;
    let periode = labels.selamanya;
    if (form.effective_from && form.effective_until) periode = t('dateRange', { from: form.effective_from, to: form.effective_until });
    else if (form.effective_from) periode = t('dateRangeFrom', { from: form.effective_from });
    else if (form.effective_until) periode = t('dateRangeUntil', { to: form.effective_until });

    return {
      product: selectedProductName || '-',
      category: catLabel,
      brand: brandLabel,
      customerGroup: cgLabel,
      store: storeLabel,
      qty: `${form.minimum_quantity} - ${qtyMax}`,
      days,
      time: timeStr,
      periode,
      method: pricingMethods.find(m => m.value === form.pricing_method)?.label || form.pricing_method,
      value: valueStr,
      type: pricingTypes.find(t => t.value === form.pricing_type)?.label || form.pricing_type
    };
  });

  function validateForm(): Record<string, string> {
    const errors: Record<string, string> = {};
    if (!form.name || !form.name.trim()) errors.name = labels.errorNameRequired;
    if (!form.product_id && !form.category_id && !form.brand_id) errors.target = labels.errorTargetRequired;
    if (form.maximum_quantity && form.minimum_quantity > Number(form.maximum_quantity)) errors.qty = labels.errorMaxQty;
    if (form.effective_from && form.effective_until && form.effective_from > form.effective_until) errors.dates = labels.errorDates;
    return errors;
  }

  function handleImportComplete() {
    fetchRules();
    toast.success(labels.importPricingRulesComplete);
  }

  async function handleBulkActivate(ids: number[]) {
    const results = await Promise.allSettled(
      ids.map(id => updatePricingRule(id, { is_active: true }))
    );
    const ok = results.filter(r => r.status === 'fulfilled' && r.value).length;
    const fail = ids.length - ok;
    if (ok > 0) toast.success(t('bulkActivateCount', { n: ok }));
    if (fail > 0) toast.error(t('bulkActivateFailCount', { n: fail }));
    fetchRules();
  }

  async function handleBulkDeactivate(ids: number[]) {
    const results = await Promise.allSettled(
      ids.map(id => updatePricingRule(id, { is_active: false }))
    );
    const ok = results.filter(r => r.status === 'fulfilled' && r.value).length;
    const fail = ids.length - ok;
    if (ok > 0) toast.success(t('bulkDeactivateCount', { n: ok }));
    if (fail > 0) toast.error(t('bulkDeactivateFailCount', { n: fail }));
    fetchRules();
  }

  async function handleBulkDelete(ids: number[]) {
    const results = await Promise.allSettled(
      ids.map(id => deletePricingRule(id))
    );
    const ok = results.filter(r => r.status === 'fulfilled' && r.value).length;
    const fail = ids.length - ok;
    if (ok > 0) toast.success(t('bulkDeleteCount', { n: ok }));
    if (fail > 0) toast.error(t('bulkDeleteFailCount', { n: fail }));
    fetchRules();
  }

  async function handleSubmitApproval(rule: PricingRule) {
    const ok = await submitPricingRule(rule.id);
    if (ok) {
      toast.success(t('ruleSubmittedApproval', { name: rule.name }));
      fetchRules();
    } else {
      toast.error(labels.failedToSubmitApproval);
    }
  }

  async function handleApprove(rule: PricingRule) {
    const ok = await approvePricingRule(rule.id);
    if (ok) {
      toast.success(t('ruleApproved', { name: rule.name }));
      fetchRules();
    } else {
      toast.error(labels.failedToApproveRule);
    }
  }

  async function handleReject(rule: PricingRule) {
    const ok = await rejectPricingRule(rule.id);
    if (ok) {
      toast.success(t('ruleRejected', { name: rule.name }));
      fetchRules();
    } else {
      toast.error(labels.failedToRejectRule);
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement || e.target instanceof HTMLSelectElement) return;
    if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
      e.preventDefault();
      if (canCreate) openAdd();
    }
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
      e.preventDefault();
      document.querySelector<HTMLInputElement>('#pricing-search')?.focus();
    }
  }

  onMount(() => {
    window.addEventListener('keydown', handleKeydown);
    fetchRules();
    return () => window.removeEventListener('keydown', handleKeydown);
  });
</script>

<div class="space-y-5">
  <a href="#pricing-rules-table" class="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-50 focus:px-4 focus:py-2 focus:rounded-lg focus:bg-primary-default focus:text-white focus:outline-none focus:ring-2 focus:ring-primary-light">
    {labels.skipToRulesTable}
  </a>

  <PricingRulesToolbar
    bind:searchQuery
    bind:approvalFilter
    bind:statusFilter
    bind:typeFilter
    bind:methodFilter
    {canCreate}
    {pricingTypes}
    {pricingMethods}
    {typeLabel}
    {methodLabel}
    oncreate={openAdd}
    onfilter={fetchRules}
    onimport={() => showImportWizard = true}
    onsimulate={() => showSimulation = true}
  />

  <div id="pricing-rules-table" class="card overflow-x-auto" aria-live="polite" aria-atomic="true">
    <span class="sr-only" aria-live="polite">
      {#if loading}{labels.loadingData}{:else if rules.length === 0}{labels.noRulesFound}{:else}{t('showingRange', { from: offset + 1, to: Math.min(offset + limit, total), total })}{/if}
    </span>
    <PricingRulesTable
      rules={sortedRules}
      {loading}
      {searchQuery}
      sortBy={sortState.sortBy}
      sortDir={sortState.sortDir}
      {canEdit}
      {canDelete}
      {canCreate}
      {pricingTypes}
      {pricingMethods}
      onsort={handleSort}
      onedit={openEdit}
      ondelete={openDelete}
      onduplicate={handleDuplicate}
      onbulkactivate={handleBulkActivate}
      onbulkdeactivate={handleBulkDeactivate}
      onbulkdelete={handleBulkDelete}
      oncreate={openAdd}
      onsubmitapproval={handleSubmitApproval}
      onapprove={handleApprove}
      onreject={handleReject}
      onrowclick={handleRowClick}
      {targetNames}
    />

    {#if !loading && rules.length > 0}
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
        <div class="flex items-center justify-end">
          <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
        </div>
      </div>
    {/if}
  </div>
</div>

<Modal bind:open={showModal} title={modalMode === 'add' ? labels.addPricingRule : labels.editPricingRule} size="xl">
  <form onsubmit={saveRule} class="space-y-5">

    <!-- Section 1: Informasi Rule -->
    <div>
      <h3 class="text-sm font-semibold text-text-primary mb-3 flex items-center gap-2">
        <span class="inline-flex items-center justify-center w-5 h-5 rounded-full bg-primary-subtle text-primary-light text-xs font-bold">1</span>
        {labels.informasiRule}
      </h3>
      <div class="space-y-3">
        <div>
          <label for="rule-name" class="block text-xs font-medium text-text-secondary mb-1">{labels.namaRule} <span class="text-danger">*</span></label>
          <Input id="rule-name" bind:value={form.name} required placeholder={labels.ruleNamePlaceholder} class="h-9 text-sm" />
          {#if showErrors && formErrors.name}
            <p class="mt-1 text-xs text-danger">{formErrors.name}</p>
          {/if}
        </div>
        <div class="grid grid-cols-4 gap-3">
          <div>
            <label for="pricing-type" class="block text-xs font-medium text-text-secondary mb-1">{labels.tipeHarga} <span class="text-danger">*</span></label>
            <select id="pricing-type" bind:value={form.pricing_type} class="w-full rounded-xl border border-border-default px-3 py-2 text-sm bg-bg-secondary text-text-primary focus:outline-none focus:ring-2 focus:ring-primary-default/30 h-9 transition-colors">
              {#each pricingTypes as pt}<option value={pt.value}>{pt.label}</option>{/each}
            </select>
            <p class="mt-0.5 text-xs leading-tight text-text-muted">{pricingTypes.find(t => t.value === form.pricing_type)?.description || ''}</p>
          </div>
          <div>
            <label for="pricing-method" class="block text-xs font-medium text-text-secondary mb-1">{labels.metode} <span class="text-danger">*</span></label>
            <select id="pricing-method" bind:value={form.pricing_method} class="w-full rounded-xl border border-border-default px-3 py-2 text-sm bg-bg-secondary text-text-primary focus:outline-none focus:ring-2 focus:ring-primary-default/30 h-9 transition-colors">
              {#each pricingMethods as pm}<option value={pm.value}>{pm.label}</option>{/each}
            </select>
            <p class="mt-0.5 text-xs leading-tight text-text-muted">{pricingMethods.find(m => m.value === form.pricing_method)?.description || ''}</p>
          </div>
          <div class="col-span-2">
            <label for="pricing-value" class="block text-xs font-medium text-text-secondary mb-1">{labels.nilai} <span class="text-danger">*</span></label>
            <div class="relative">
              {#if getMethodConfig(form.pricing_method).prefix}
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-sm text-text-muted pointer-events-none select-none">{getMethodConfig(form.pricing_method).prefix}</span>
              {/if}
              <Input
                id="pricing-value"
                type="number"
                bind:value={form.pricing_value}
                required
                min="0"
                step="0.01"
                placeholder={getMethodConfig(form.pricing_method).placeholder}
                class="h-9 text-sm {getMethodConfig(form.pricing_method).prefix ? 'pl-7' : ''} {getMethodConfig(form.pricing_method).suffix ? 'pr-7' : ''}"
              />
              {#if getMethodConfig(form.pricing_method).suffix}
                <span class="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-text-muted pointer-events-none select-none">{getMethodConfig(form.pricing_method).suffix}</span>
              {/if}
            </div>
            <p class="mt-0.5 text-xs leading-tight text-text-muted">{getMethodConfig(form.pricing_method).helper}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Section 2: Kondisi -->
    <div class="border-t border-border-default pt-4">
      <h3 class="text-sm font-semibold text-text-primary mb-3 flex items-center gap-2">
        <span class="inline-flex items-center justify-center w-5 h-5 rounded-full bg-primary-subtle text-primary-light text-xs font-bold">2</span>
        {labels.kondisi}
      </h3>
      <div class="grid grid-cols-4 gap-3">
        <div>
          <label for="min-qty" class="block text-xs font-medium text-text-secondary mb-1">{labels.minQty}</label>
          <Input id="min-qty" type="number" bind:value={form.minimum_quantity} min="1" placeholder="1" class="h-9 text-sm" />
        </div>
        <div>
          <label for="max-qty" class="block text-xs font-medium text-text-secondary mb-1">{labels.maxQty}</label>
          <Input id="max-qty" type="number" bind:value={form.maximum_quantity} min="1" placeholder={labels.tanpaBatas} class="h-9 text-sm" />
          <p class="mt-0.5 text-xs leading-tight text-text-muted">{labels.emptyMeansUnlimited}</p>
          {#if showErrors && formErrors.qty}
            <p class="mt-0.5 text-xs text-danger">{formErrors.qty}</p>
          {/if}
        </div>
        <div>
          <label for="customer-group" class="block text-xs font-medium text-text-secondary mb-1">{labels.customerGroup}</label>
          <select id="customer-group" bind:value={form.customer_group_id} class="w-full rounded-xl border border-border-default px-3 py-2 text-sm bg-bg-secondary text-text-primary focus:outline-none focus:ring-2 focus:ring-primary-default/30 h-9 transition-colors">
            <option value={null}>{labels.semuaGroup}</option>
            {#each customerGroups as cg}<option value={cg.id}>{cg.name}</option>{/each}
          </select>
          <p class="mt-0.5 text-xs leading-tight text-text-muted">{labels.semuaGroupHint}</p>
        </div>
        <div>
          <label for="store-id" class="block text-xs font-medium text-text-secondary mb-1">{labels.outlet}</label>
          <select id="store-id" bind:value={form.store_id} class="w-full rounded-xl border border-border-default px-3 py-2 text-sm bg-bg-secondary text-text-primary focus:outline-none focus:ring-2 focus:ring-primary-default/30 h-9 transition-colors">
            <option value={null}>{labels.semuaOutlet}</option>
            {#each stores as s}<option value={s.id}>{s.name}</option>{/each}
          </select>
          <p class="mt-0.5 text-xs leading-tight text-text-muted">{labels.semuaOutletHint}</p>
        </div>
      </div>
    </div>

    <!-- Section 3: Target -->
    <div class="border-t border-border-default pt-4">
      <h3 class="text-sm font-semibold text-text-primary mb-3 flex items-center gap-2">
        <span class="inline-flex items-center justify-center w-5 h-5 rounded-full bg-primary-subtle text-primary-light text-xs font-bold">3</span>
        {labels.target}
      </h3>
      <div class="grid grid-cols-3 gap-3">
        <div>
          <label for="product-search" class="block text-xs font-medium text-text-secondary mb-1">{labels.produk}</label>
          {#if selectedProductName}
            <div class="flex items-center gap-2 h-9 px-3 rounded-xl border border-border-default bg-bg-secondary text-sm">
              <span class="flex-1 truncate text-text-primary">{selectedProductName}</span>
              <button type="button" onclick={clearProduct} class="text-text-muted hover:text-danger text-xs shrink-0">x</button>
            </div>
          {:else}
            <div class="relative">
              <Input id="product-search" bind:value={productSearchQuery} oninput={handleProductSearch} placeholder={labels.searchProducts} class="h-9 text-sm" />
              {#if productSearchResults.length > 0}
                <div class="absolute z-50 mt-1 w-full bg-surface-default border border-border rounded-xl shadow-xl max-h-48 overflow-auto">
                  {#each productSearchResults as p}
                    <button type="button" onclick={() => selectProduct(p)} class="w-full text-left px-3 py-2 text-sm hover:bg-surface-hover truncate">
                      <span class="text-text-primary">{p.name}</span> <span class="text-text-muted">({p.sku})</span>
                    </button>
                  {/each}
                </div>
              {/if}
            </div>
          {/if}
        </div>
        <div>
          <label for="category-search" class="block text-xs font-medium text-text-secondary mb-1">{labels.kategori}</label>
          {#if selectedCategoryName}
            <div class="flex items-center gap-2 h-9 px-3 rounded-xl border border-border-default bg-bg-secondary text-sm">
              <span class="flex-1 truncate text-text-primary">{selectedCategoryName}</span>
              <button type="button" onclick={clearCategory} class="text-text-muted hover:text-danger text-xs shrink-0">x</button>
            </div>
          {:else}
            <div class="relative">
              <Input id="category-search" bind:value={categorySearchQuery} oninput={handleCategorySearch} placeholder={labels.searchCategories} class="h-9 text-sm" />
              {#if categorySearchResults.length > 0}
                <div class="absolute z-50 mt-1 w-full bg-surface-default border border-border rounded-xl shadow-xl max-h-48 overflow-auto">
                  {#each categorySearchResults as c}
                    <button type="button" onclick={() => selectCategory(c)} class="w-full text-left px-3 py-2 text-sm hover:bg-surface-hover truncate">
                      <span class="text-text-primary">{c.name}</span>
                    </button>
                  {/each}
                </div>
              {/if}
            </div>
          {/if}
        </div>
        <div>
          <label for="brand-search" class="block text-xs font-medium text-text-secondary mb-1">{labels.brand}</label>
          {#if selectedBrandName}
            <div class="flex items-center gap-2 h-9 px-3 rounded-xl border border-border-default bg-bg-secondary text-sm">
              <span class="flex-1 truncate text-text-primary">{selectedBrandName}</span>
              <button type="button" onclick={clearBrand} class="text-text-muted hover:text-danger text-xs shrink-0">x</button>
            </div>
          {:else}
            <div class="relative">
              <Input id="brand-search" bind:value={brandSearchQuery} oninput={handleBrandSearch} placeholder={labels.searchBrands} class="h-9 text-sm" />
              {#if brandSearchResults.length > 0}
                <div class="absolute z-50 mt-1 w-full bg-surface-default border border-border rounded-xl shadow-xl max-h-48 overflow-auto">
                  {#each brandSearchResults as b}
                    <button type="button" onclick={() => selectBrand(b)} class="w-full text-left px-3 py-2 text-sm hover:bg-surface-hover truncate">
                      <span class="text-text-primary">{b.name}</span>
                    </button>
                  {/each}
                </div>
              {/if}
            </div>
          {/if}
        </div>
      </div>
      <p class="mt-1.5 text-xs text-text-muted">{labels.emptyUnusedFields}</p>
      {#if showErrors && formErrors.target}
        <p class="mt-0.5 text-xs text-danger">{formErrors.target}</p>
      {/if}
    </div>

    <!-- Section 4: Jadwal -->
    <div class="border-t border-border-default pt-4">
      <h3 class="text-sm font-semibold text-text-primary mb-3 flex items-center gap-2">
        <span class="inline-flex items-center justify-center w-5 h-5 rounded-full bg-primary-subtle text-primary-light text-xs font-bold">4</span>
        {labels.jadwal}
      </h3>
      <div class="space-y-3">
        <div class="flex items-center gap-2">
          <button type="button" onclick={allDaysSelected ? clearDays : selectAllDays}
            class="h-6 px-2.5 rounded-md text-xs font-medium transition-all {allDaysSelected ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'bg-bg-secondary text-text-muted border border-border-default hover:bg-surface-hover'}">
            {labels.semuaHari}
          </button>
          <button type="button" onclick={workDaysSelected ? clearDays : selectWorkDays}
            class="h-6 px-2.5 rounded-md text-xs font-medium transition-all {workDaysSelected ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'bg-bg-secondary text-text-muted border border-border-default hover:bg-surface-hover'}">
            {labels.hariKerja}
          </button>
          <button type="button" onclick={weekendSelected ? clearDays : selectWeekend}
            class="h-6 px-2.5 rounded-md text-xs font-medium transition-all {weekendSelected ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'bg-bg-secondary text-text-muted border border-border-default hover:bg-surface-hover'}">
            {labels.weekend}
          </button>
        </div>
        <div class="flex flex-wrap gap-1">
          {#each dayOptions as day}
            <button type="button" onclick={() => toggleDay(day.value)}
              class="h-7 px-2.5 rounded-lg text-xs font-medium transition-all {form.recurrence_days.includes(day.value) ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'bg-bg-secondary text-text-muted border border-border-default hover:bg-surface-hover'}">
              {day.label}
            </button>
          {/each}
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="time-from" class="block text-xs font-medium text-text-secondary mb-1">{labels.dariJam}</label>
            <Input id="time-from" type="time" bind:value={form.time_from} class="h-9 text-sm" />
          </div>
          <div>
            <label for="time-to" class="block text-xs font-medium text-text-secondary mb-1">{labels.sampaiJam}</label>
            <Input id="time-to" type="time" bind:value={form.time_to} class="h-9 text-sm" />
          </div>
        </div>
        {#if !form.time_from && !form.time_to}
          <p class="text-xs leading-tight text-text-muted">{labels.emptyMeansAllDay}</p>
        {/if}
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="effective-from" class="block text-xs font-medium text-text-secondary mb-1">{labels.berlakuDari}</label>
            <Input id="effective-from" type="date" bind:value={form.effective_from} class="h-9 text-sm" />
          </div>
          <div>
            <label for="effective-until" class="block text-xs font-medium text-text-secondary mb-1">{labels.berlakuSampai}</label>
            <Input id="effective-until" type="date" bind:value={form.effective_until} class="h-9 text-sm" />
          </div>
        </div>
        {#if !form.effective_from && !form.effective_until}
          <p class="text-xs leading-tight text-text-muted">{labels.emptyMeansForever}</p>
        {/if}
        {#if showErrors && formErrors.dates}
          <p class="text-xs text-danger">{formErrors.dates}</p>
        {/if}
      </div>
    </div>

    <!-- Section 5: Ringkasan -->
    <div class="border-t border-border-default pt-4">
      <h3 class="text-sm font-semibold text-text-primary mb-3 flex items-center gap-2">
        <span class="inline-flex items-center justify-center w-5 h-5 rounded-full bg-primary-subtle text-primary-light text-xs font-bold">5</span>
        {labels.ringkasanRule}
      </h3>
      <div class="bg-bg-secondary/60 border border-border-default rounded-xl p-4">
        <div class="grid grid-cols-2 gap-x-6 gap-y-2 text-xs">
          <div class="flex items-center gap-2">
            <span class="text-text-muted w-20 shrink-0">{labels.tipe}</span>
            <Badge variant="primary" size="sm">{summaryPreview.type}</Badge>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-text-muted w-20 shrink-0">{labels.metode}</span>
            <span class="text-text-primary font-medium">{summaryPreview.method}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-text-muted w-20 shrink-0">{labels.nilai}</span>
            <span class="text-primary-light font-semibold">{summaryPreview.value}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-text-muted w-20 shrink-0">{labels.qty}</span>
            <span class="text-text-primary">{summaryPreview.qty}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-text-muted w-20 shrink-0">{labels.produk}</span>
            <span class="text-text-primary truncate">{summaryPreview.product}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-text-muted w-20 shrink-0">{labels.kategori}</span>
            <span class="text-text-primary">{summaryPreview.category}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-text-muted w-20 shrink-0">{labels.brand}</span>
            <span class="text-text-primary">{summaryPreview.brand}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-text-muted w-20 shrink-0">{labels.customer}</span>
            <span class="text-text-primary">{summaryPreview.customerGroup}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-text-muted w-20 shrink-0">{labels.outlet}</span>
            <span class="text-text-primary">{summaryPreview.store}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-text-muted w-20 shrink-0">{labels.hari}</span>
            <span class="text-text-primary">{summaryPreview.days}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-text-muted w-20 shrink-0">{labels.jam}</span>
            <span class="text-text-primary">{summaryPreview.time}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-text-muted w-20 shrink-0">{labels.periode}</span>
            <span class="text-text-primary">{summaryPreview.periode}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Opsi Tambahan -->
    <div class="flex items-center gap-5">
      <label class="flex items-center gap-2 cursor-pointer group">
        <input type="checkbox" bind:checked={form.allow_combine} class="rounded border-border-default text-primary-default focus:ring-primary-default/30" />
        <span class="text-sm text-text-secondary group-hover:text-text-primary transition-colors">{labels.allowCombineLabel}</span>
      </label>
      {#if modalMode === 'edit'}
        <label class="flex items-center gap-2 cursor-pointer group">
          <input type="checkbox" bind:checked={form.is_active} class="rounded border-border-default text-primary-default focus:ring-primary-default/30" />
          <span class="text-sm text-text-secondary group-hover:text-text-primary transition-colors">{labels.aktif}</span>
        </label>
      {/if}
    </div>

    {#if showConflictWarning && conflictRules.length > 0}
      <div class="rounded-xl border border-warning/40 bg-warning-subtle/20 p-4 space-y-3">
        <div class="flex items-center gap-2">
          <AlertTriangle size={18} class="text-warning shrink-0" />
          <span class="text-sm font-semibold text-text-primary">{labels.conflictFound}</span>
        </div>
        <p class="text-xs text-text-muted">{t('conflictCount', { count: conflictRules.length })}</p>
        <div class="space-y-1.5 max-h-32 overflow-y-auto">
          {#each conflictRules as c}
            <div class="flex items-center gap-2 text-xs bg-surface-default/60 rounded-lg px-3 py-1.5">
              <span class="font-medium text-text-primary">{c.name}</span>
              <span class="text-text-muted">|</span>
              <span class="text-text-muted">{c.pricing_method}</span>
              <span class="text-text-muted">|</span>
              <span class="text-text-muted">{t('priorityLabel', { value: c.priority })}</span>
            </div>
          {/each}
        </div>
        <p class="text-xs text-text-muted">{labels.conflictExplanation}</p>
      </div>
    {/if}

  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => { showModal = false; showConflictWarning = false; conflictRules = []; }} disabled={saving}>{labels.cancel}</Button>
    <Button variant="primary" class="min-w-32" onclick={saveRule} disabled={saving}>
      {#if saving}<Loader2 class="w-4 h-4 mr-2 animate-spin" />{/if}
      {showConflictWarning ? labels.keepSaving : modalMode === 'add' ? labels.buatRule : labels.updateRule}
    </Button>
  {/snippet}
</Modal>

<ConfirmDeleteModal bind:open={showDeleteModal} onconfirm={confirmDelete} />

<ImportWizard
  bind:open={showImportWizard}
  module="pricing_rules"
  displayName={labels.pricingRules}
  onComplete={handleImportComplete}
/>

<PriceSimulationModal bind:open={showSimulation} />

<PricingRuleDetailDrawer
  bind:open={showDetailDrawer}
  rule={detailDrawerRule}
  {canEdit}
  {canDelete}
  {targetNames}
  {customerGroups}
  {stores}
  onclose={handleDetailDrawerClose}
  onedit={(rule) => { showDetailDrawer = false; openEdit(rule); }}
  ondelete={(rule) => { showDetailDrawer = false; openDelete(rule); }}
/>
