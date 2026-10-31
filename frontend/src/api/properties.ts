import api from './client';

export type PropertyType = 'residential' | 'commercial' | 'land' | 'other';

export interface Property {
  id: string;
  user_id: string;
  name: string;
  property_type: PropertyType;
  location: string;
  /** incl. stamp duty and registration */
  purchase_price: number;
  purchase_date: string;
  /** full property value */
  current_value: number;
  value_updated_at: string;
  ownership_pct: number;
  home_loan_id: string;
  is_self_occupied: boolean;
  monthly_rent: number;
  /** maintenance, property tax */
  annual_costs: number;
  notes: string;
  created_at: string;
  updated_at: string;
  owned_value: number;
  gain_loss: number;
  cagr: number;
  /** % of current value */
  net_rental_yield: number;
  loan_outstanding: number;
  /** owned value - loan outstanding */
  equity: number;
}

export interface PropertyRequest {
  name: string;
  property_type: PropertyType;
  location: string;
  purchase_price: number;
  purchase_date: string;
  current_value: number;
  value_updated_at: string;
  ownership_pct: number;
  home_loan_id: string;
  is_self_occupied: boolean;
  monthly_rent: number;
  annual_costs: number;
  notes: string;
}

export const PROPERTY_TYPE_LABELS: Record<PropertyType, string> = {
  residential: 'Residential',
  commercial: 'Commercial',
  land: 'Land',
  other: 'Other',
};

export const getProperties = () => api.get<Property[]>('/properties');
export const getProperty = (id: string) => api.get<Property>(`/properties/${id}`);
export const createProperty = (data: PropertyRequest) => api.post<Property>('/properties', data);
export const updateProperty = (id: string, data: PropertyRequest) => api.put<Property>(`/properties/${id}`, data);
export const deleteProperty = (id: string) => api.delete(`/properties/${id}`);
