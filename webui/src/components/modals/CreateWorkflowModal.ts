import { store } from '@/store';
import { workflowsAPI } from '@/api/workflows';
import type { CreateWorkflowRequest, ActivityType, TriggerKind } from '@/types';

export function CreateWorkflowModal(): HTMLElement {
  const container = document.createElement('div');
  container.className = 'p-modal';
  container.innerHTML = `
    <div class="p-modal__dialog" role="dialog">
      <header class="p-modal__header">
        <h2 class="p-modal__title">Create Workflow</h2>
        <button class="p-modal__close-button" aria-label="Close"></button>
      </header>
      <div class="p-modal__body">
        <form id="create-workflow-form">
          <div class="form-group">
            <label for="name">Name *</label>
            <input type="text" id="name" name="name" required />
          </div>

          <div class="form-group">
            <label for="description">Description</label>
            <textarea id="description" name="description"></textarea>
          </div>

          <div class="form-group">
            <label for="workflow_type">Workflow Type *</label>
            <input type="text" id="workflow_type" name="workflow_type" required />
          </div>

          <div class="form-group">
            <label for="activity">Activity *</label>
            <select id="activity" name="activity" required>
              <option value="">Select an activity</option>
              <option value="reboot">Reboot</option>
              <option value="install_package">Install Package</option>
              <option value="upgrade_package">Upgrade Package</option>
              <option value="remove_package">Remove Package</option>
              <option value="apply_config">Apply Config</option>
              <option value="validate_config">Validate Config</option>
              <option value="run_script">Run Script</option>
            </select>
          </div>

          <div class="form-group">
            <label for="target_filter">Target Filter *</label>
            <input type="text" id="target_filter" name="target_filter" placeholder="e.g., os=ubuntu" required />
          </div>

          <div class="form-group">
            <label for="trigger_kind">Trigger Kind *</label>
            <select id="trigger_kind" name="trigger_kind" required>
              <option value="">Select a trigger</option>
              <option value="manual">Manual</option>
              <option value="scheduled">Scheduled</option>
              <option value="event">Event</option>
              <option value="state_change">State Change</option>
            </select>
          </div>

          <div class="form-group" id="cron-group" style="display: none;">
            <label for="cron">Cron Schedule</label>
            <input type="text" id="cron" name="cron" placeholder="0 0 * * *" />
          </div>

          <div class="form-group">
            <label for="success_threshold">Success Threshold (%) *</label>
            <input type="number" id="success_threshold" name="success_threshold" min="0" max="100" value="100" required />
          </div>

          <div class="form-group">
            <label for="loop_threshold_ms">Loop Threshold (ms) *</label>
            <input type="number" id="loop_threshold_ms" name="loop_threshold_ms" min="0" value="0" required />
          </div>

          <div class="form-group">
            <label for="timeout_ms">Timeout (ms) *</label>
            <input type="number" id="timeout_ms" name="timeout_ms" min="0" value="3600000" required />
          </div>

          <div class="form-group">
            <label>
              <input type="checkbox" id="active" name="active" checked />
              Active
            </label>
          </div>
        </form>
      </div>
      <footer class="p-modal__footer">
        <button class="p-button p-button--base cancel-btn">Cancel</button>
        <button class="p-button p-button--positive submit-btn">Create</button>
      </footer>
    </div>
  `;

  const form = container.querySelector('#create-workflow-form') as HTMLFormElement;
  const submitBtn = container.querySelector('.submit-btn') as HTMLButtonElement;
  const cancelBtn = container.querySelector('.cancel-btn') as HTMLButtonElement;
  const closeBtn = container.querySelector('.p-modal__close-button') as HTMLButtonElement;
  const triggerKindSelect = container.querySelector('#trigger_kind') as HTMLSelectElement;
  const cronGroup = container.querySelector('#cron-group') as HTMLElement;

  triggerKindSelect.addEventListener('change', (e) => {
    if ((e.target as HTMLSelectElement).value === 'scheduled') {
      cronGroup.style.display = 'block';
    } else {
      cronGroup.style.display = 'none';
    }
  });

  const handleClose = () => {
    store.closeModal('create-workflow');
  };

  submitBtn.addEventListener('click', async (e) => {
    e.preventDefault();
    if (!form.checkValidity()) {
      form.reportValidity();
      return;
    }

    try {
      submitBtn.disabled = true;
      const formData = new FormData(form);

      const triggerKind = formData.get('trigger_kind') as TriggerKind;
      const trigger: any = { kind: triggerKind };

      if (triggerKind === 'scheduled') {
        trigger.cron = formData.get('cron');
      }

      const request: CreateWorkflowRequest = {
        name: formData.get('name') as string,
        description: formData.get('description') as string,
        workflow_type: formData.get('workflow_type') as string,
        activity: formData.get('activity') as ActivityType,
        target_filter: formData.get('target_filter') as string,
        trigger,
        success_threshold: parseInt(formData.get('success_threshold') as string),
        loop_threshold_ms: parseInt(formData.get('loop_threshold_ms') as string),
        timeout_ms: parseInt(formData.get('timeout_ms') as string),
        active: (formData.get('active') as any) === 'on',
        params: {},
      };

      await workflowsAPI.create(request);
      store.showToast('Workflow created successfully', 'success');
      store.setWorkflowLoading(true);
      const workflows = await workflowsAPI.getAll();
      store.setWorkflows(workflows);
      handleClose();
    } catch (err) {
      store.showToast('Failed to create workflow', 'error');
      submitBtn.disabled = false;
    }
  });

  cancelBtn.addEventListener('click', handleClose);
  closeBtn.addEventListener('click', handleClose);

  return container;
}
