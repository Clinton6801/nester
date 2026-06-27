import { apiRequest } from "@/lib/api/client";

export interface ProjectionPoint {
  month: number;
  principal: string;
  yield: string;
  total: string;
}

export interface ProjectionSummary {
  total_deposited: string;
  total_yield: string;
  final_balance: string;
  effective_apy: string;
}

export interface ProjectionInput {
  initial_deposit: string;
  monthly_contribution: string;
  apy: string;
  period_months: number;
  compound_frequency: "daily" | "monthly";
}

export interface ProjectionOutput {
  vault_id?: string;
  currency: string;
  current_apy: number;
  input: ProjectionInput;
  timeline: ProjectionPoint[];
  summary: ProjectionSummary;
  calculated_at: string;
}

export interface VaultProjectionParams {
  deposit: string;
  period: string; // e.g., "12m"
  compound?: "daily" | "monthly";
  apy?: string; // Optional APY override
}

export const projectionApi = {
  // Generic projection calculation
  calculateProjection: (input: ProjectionInput) =>
    apiRequest<ProjectionOutput>("/tools/projection", {
      method: "POST",
      body: JSON.stringify(input),
    }),

  // Vault-specific projection
  calculateVaultProjection: (vaultId: string, params: VaultProjectionParams) => {
    const query = new URLSearchParams({
      deposit: params.deposit,
      period: params.period,
      ...(params.compound && { compound: params.compound }),
      ...(params.apy && { apy: params.apy }),
    });

    return apiRequest<ProjectionOutput>(`/vaults/${vaultId}/projection?${query}`);
  },
};

// Helper functions for working with projection data

export function formatProjectionAmount(amount: string): string {
  const num = parseFloat(amount);
  return num.toLocaleString("en-US", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

export function formatProjectionAPY(apy: string | number): string {
  const num = typeof apy === "string" ? parseFloat(apy) : apy;
  return (num * 100).toFixed(2) + "%";
}

export function calculateMonthlyGrowthRate(timeline: ProjectionPoint[]): number {
  if (timeline.length < 2) return 0;
  
  const firstMonth = parseFloat(timeline[0].total);
  const lastMonth = parseFloat(timeline[timeline.length - 1].total);
  const months = timeline.length;
  
  if (firstMonth === 0) return 0;
  
  return Math.pow(lastMonth / firstMonth, 1 / months) - 1;
}

export function getProjectionMilestones(timeline: ProjectionPoint[]): ProjectionPoint[] {
  // Return milestone points (every 3 months or at key intervals)
  const milestones: ProjectionPoint[] = [];
  
  timeline.forEach((point, index) => {
    // Always include first and last
    if (index === 0 || index === timeline.length - 1) {
      milestones.push(point);
      return;
    }
    
    // Include every 3rd month for shorter timelines, every 6th for longer
    const interval = timeline.length > 24 ? 6 : 3;
    if ((point.month - 1) % interval === 0) {
      milestones.push(point);
    }
  });
  
  return milestones;
}