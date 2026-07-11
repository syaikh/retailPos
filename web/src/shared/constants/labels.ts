/**
 * Centralized i18n labels for the application.
 * All user-facing strings should reference this file.
 * Currently supports Indonesian (primary) with English fallbacks.
 */

export const labels = {
  // Actions
  save: 'Simpan',
  cancel: 'Batal',
  delete: 'Hapus',
  edit: 'Edit',
  create: 'Buat',
  close: 'Tutup',
  confirm: 'Konfirmasi',
  back: 'Kembali',
  apply: 'Terapkan',
  reset: 'Atur Ulang',
  search: 'Cari...',
  clearAll: 'Hapus Semua',
  loading: 'Memuat...',
  saving: 'Menyimpan...',
  deleting: 'Menghapus...',

  // Auth
  login: 'Masuk',
  logout: 'Keluar',
  username: 'Username',
  password: 'Password',

  // Dashboard
  dashboard: 'Dashboard',
  todayRevenue: 'Pendapatan Hari Ini',
  todaySales: 'Penjualan Hari Ini',
  totalProducts: 'Total Produk',
  lowStock: 'Stok Rendah',

  // POS
  pointOfSale: 'Point of Sale',
  cashReceived: 'Uang Diterima',
  change: 'Uang Kembali',
  exact: 'Tepat',
  clearCart: 'Kosongkan Keranjang',
  checkout: 'Checkout',
  paymentMethod: 'Metode Pembayaran',
  cash: 'Tunai',
  card: 'Kartu',
  eWallet: 'E-Wallet',

  // Products
  products: 'Produk',
  productName: 'Nama Produk',
  price: 'Harga',
  cost: 'Modal',
  stock: 'Stok',
  category: 'Kategori',
  brand: 'Brand',
  unitOfMeasure: 'Satuan',
  status: 'Status',
  active: 'Aktif',
  inactive: 'Tidak Aktif',

  // Customers
  customers: 'Pelanggan',
  name: 'Nama',
  phone: 'Telepon',
  email: 'Email',
  address: 'Alamat',

  // Transactions
  transactions: 'Riwayat Transaksi',
  invoice: 'Invoice',
  date: 'Tanggal',
  total: 'Total',
  payment: 'Pembayaran',

  // Admin
  users: 'Pengguna',
  roles: 'Peran',
  auditLogs: 'Log Audit',
  permissions: 'Izin',

  // Settings
  categories: 'Kategori',
  brands: 'Brand',
  unitsOfMeasure: 'Satuan',

  // Filters
  filterBy: 'Filter',
  allSemua: 'Semua',
  last24Hours: '24 Jam Terakhir',
  last7Days: '7 Hari Terakhir',
  last30Days: '30 Hari Terakhir',
  last90Days: '90 Hari Terakhir',
  thisMonth: 'Bulan Ini',
  thisYear: 'Tahun Ini',
  today: 'Hari Ini',
  yesterday: 'Kemarin',
  customRange: 'Rentang Kustom',

  // Pagination
  showing: 'Menampilkan',
  of: 'dari',
  rows: 'baris',
  previous: 'Sebelumnya',
  next: 'Selanjutnya',

  // Empty states
  noResults: 'Tidak ada hasil',
  noData: 'Belum ada data',
  noProducts: 'Belum ada produk',
  noCustomers: 'Belum ada pelanggan',
  noTransactions: 'Belum ada transaksi',

  // Delete confirmations
  deleteConfirm: 'Apakah Anda yakin ingin menghapus?',
  deletePermanent: 'Data akan dihapus secara permanen dan tidak dapat dikembalikan.',

  // Errors
  errorOccurred: 'Terjadi kesalahan',
  accessDenied: 'Akses ditolak',
  notFound: 'Halaman tidak ditemukan',
  failedToLoad: 'Gagal memuat data',
  tryAgain: 'Coba lagi',
} as const;

export type LabelKey = keyof typeof labels;
