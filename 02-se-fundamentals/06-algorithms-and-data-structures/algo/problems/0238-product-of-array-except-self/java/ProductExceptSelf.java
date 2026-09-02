public class ProductExceptSelf {
    // 前缀积×后缀积两趟遍历:res 先存左侧乘积,再原地乘右侧乘积。
    // 不用除法(含 0 会崩)。O(n) 时间,除输出外 O(1) 空间。
    public static int[] productExceptSelf(int[] nums) {
        int n = nums.length;
        int[] res = new int[n];
        int prefix = 1;
        for (int i = 0; i < n; i++) {
            res[i] = prefix;
            prefix *= nums[i];
        }
        int suffix = 1;
        for (int i = n - 1; i >= 0; i--) {
            res[i] *= suffix;
            suffix *= nums[i];
        }
        return res;
    }
}
