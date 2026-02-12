"use client";

import React, { useState } from "react";
import { FormSchema } from "@/lib/ui-schema";
import { Card } from "@/components/ui/card";
import { apiFetch } from "@/lib/api";

interface FormRendererProps {
  schema: FormSchema;
}

export function FormRenderer({ schema }: FormRendererProps) {
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(false);

    try {
      const action = schema.submitAction;
      await apiFetch(action.action || '', {
        method: action.method || 'POST',
        body: JSON.stringify(formData),
      });
      
      setSuccess(true);
      setFormData({});
      
      // Redirect or callback if needed
      if (action.href) {
        window.location.href = action.href;
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to submit form");
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (name: string, value: any) => {
    setFormData({ ...formData, [name]: value });
  };

  return (
    <Card className="bg-navy border-blue p-6">
      <form onSubmit={handleSubmit} className="space-y-6">
        <div>
          <h2 className="text-2xl font-bold text-orange mb-2">{schema.title}</h2>
          {schema.description && (
            <p className="text-white/60">{schema.description}</p>
          )}
        </div>

        {error && (
          <div className="bg-red-900/20 border border-red-500 text-red-400 p-4 rounded">
            {error}
          </div>
        )}

        {success && (
          <div className="bg-green-900/20 border border-green-500 text-green-400 p-4 rounded">
            Form submitted successfully!
          </div>
        )}

        <div className="space-y-4">
          {schema.fields.map((field) => (
            <div key={field.name}>
              <label className="block text-white mb-2">
                {field.label}
                {field.required && <span className="text-red-400 ml-1">*</span>}
              </label>
              
              {field.type === 'textarea' ? (
                <textarea
                  name={field.name}
                  value={formData[field.name] || field.defaultValue || ''}
                  onChange={(e) => handleChange(field.name, e.target.value)}
                  placeholder={field.placeholder}
                  required={field.required}
                  disabled={field.disabled || loading}
                  className="w-full bg-navy border border-blue rounded px-3 py-2 text-white focus:border-orange"
                  rows={4}
                />
              ) : field.type === 'select' ? (
                <select
                  name={field.name}
                  value={formData[field.name] || field.defaultValue || ''}
                  onChange={(e) => handleChange(field.name, e.target.value)}
                  required={field.required}
                  disabled={field.disabled || loading}
                  className="w-full bg-navy border border-blue rounded px-3 py-2 text-white focus:border-orange"
                >
                  <option value="">Select {field.label}</option>
                  {field.options?.map((opt) => (
                    <option key={opt.value} value={opt.value}>{opt.label}</option>
                  ))}
                </select>
              ) : field.type === 'checkbox' ? (
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    name={field.name}
                    checked={formData[field.name] || field.defaultValue || false}
                    onChange={(e) => handleChange(field.name, e.target.checked)}
                    disabled={field.disabled || loading}
                    className="mr-2"
                  />
                  <span className="text-white/60">{field.placeholder}</span>
                </div>
              ) : (
                <input
                  type={field.type}
                  name={field.name}
                  value={formData[field.name] || field.defaultValue || ''}
                  onChange={(e) => handleChange(field.name, e.target.value)}
                  placeholder={field.placeholder}
                  required={field.required}
                  disabled={field.disabled || loading}
                  className="w-full bg-navy border border-blue rounded px-3 py-2 text-white focus:border-orange"
                />
              )}
              
              {field.validation?.message && (
                <p className="text-xs text-white/40 mt-1">{field.validation.message}</p>
              )}
            </div>
          ))}
        </div>

        <div className="flex gap-3">
          <button
            type="submit"
            disabled={loading}
            className="bg-orange hover:bg-orange/80 text-white px-6 py-2 rounded transition-colors disabled:opacity-50"
          >
            {loading ? 'Submitting...' : schema.submitAction.label}
          </button>
          
          {schema.cancelAction && (
            <button
              type="button"
              onClick={() => window.history.back()}
              className="bg-transparent hover:bg-blue/20 text-white px-6 py-2 rounded border border-blue transition-colors"
            >
              {schema.cancelAction.label}
            </button>
          )}
        </div>
      </form>
    </Card>
  );
}
